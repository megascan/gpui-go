//go:build windows

package windows

import (
	"errors"
	"fmt"
	"image"
	"log/slog"
	"math/bits"
	"sync"
	"syscall"
	"unsafe"

	xwin "golang.org/x/sys/windows"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/com"
	"github.com/megascan/gpui-go/internal/win/d3d"
	"github.com/megascan/gpui-go/internal/win/dwrite"
	"github.com/megascan/gpui-go/internal/win/win32"
	"github.com/megascan/gpui-go/render"
)

// DISABLE_DIRECT_COMPOSITION is the environment variable that turns DirectComposition off.
const DISABLE_DIRECT_COMPOSITION = "GPUI_DISABLE_DIRECT_COMPOSITION"

const (
	renderTargetFormat    = d3d.DXGI_FORMAT_B8G8R8A8_UNORM
	maxInstanceBufferSize = 256 * 1024 * 1024
	bufferCount           = 3
)

type directXFontInfo struct {
	gammaRatios               [4]float32
	grayscaleEnhancedContrast float32
	subpixelEnhancedContrast  float32
	isBgr                     bool
}

var _ render.Renderer = (*DirectXRenderer)(nil)

// DirectXRenderer draws scenes into a window with Direct3D 11, presenting through a flip-model
// swap chain, composed by DirectComposition unless it is disabled.
type DirectXRenderer struct {
	hwnd              win32.HWND
	atlas             *DirectXAtlas
	devices           *DirectXRendererDevices
	resources         *directXResources
	globals           *directXGlobalElements
	pipelines         *directXRenderPipelines
	directComposition *directComposition
	fontInfo          *directXFontInfo

	width  uint32
	height uint32

	skipDraws  bool
	background gpui.WindowBackgroundAppearance
}

// DirectXRendererDevices are the Direct3D objects a renderer draws with; each field is one
// counted reference.
type DirectXRendererDevices struct {
	Adapter        *d3d.IDXGIAdapter1
	DxgiFactory    *d3d.IDXGIFactory6
	Device         *d3d.ID3D11Device
	DeviceContext  *d3d.ID3D11DeviceContext
	deviceContext1 *d3d.ID3D11DeviceContext1
	dxgiDevice     *d3d.IDXGIDevice
	annotation     *d3d.ID3DUserDefinedAnnotation
}

type directXResources struct {
	swapChain           *d3d.IDXGISwapChain1
	renderTarget        *d3d.ID3D11Texture2D
	renderTargetView    *d3d.ID3D11RenderTargetView
	depthStencilTexture *d3d.ID3D11Texture2D
	depthStencilView    *d3d.ID3D11DepthStencilView
	viewport            d3d.D3D11_VIEWPORT
}

type directXRenderPipelines struct {
	shadowPipeline     *pipelineState[gpui.Shadow]
	quadPipeline       *pipelineState[gpui.Quad]
	opaqueQuadPipeline *opaqueQuadPipeline
	pathTilePipeline   *pipelineState[gpui.PathTile]
	pathCurves         *structuredBuffer[gpui.PathCurve]
	pathPaints         *structuredBuffer[gpui.PathPaint]
	underlinePipeline  *pipelineState[gpui.Underline]
	monoSprites        *pipelineState[gpui.MonochromeSprite]
	subpixelSprites    *pipelineState[gpui.SubpixelSprite]
	polySprites        *pipelineState[gpui.PolychromeSprite]
}

type directXGlobalElements struct {
	globalParamsBuffer *d3d.ID3D11Buffer
	batchParamsBuffer  *d3d.ID3D11Buffer
	sampler            *d3d.ID3D11SamplerState
}

type annotation struct {
	inner *d3d.ID3DUserDefinedAnnotation
}

func newAnnotation(a *d3d.ID3DUserDefinedAnnotation, label string) annotation {
	name, err := xwin.UTF16PtrFromString(label)
	if err != nil {
		name, _ = xwin.UTF16PtrFromString("")
	}
	a.BeginEvent(name)
	return annotation{inner: a}
}

func (a annotation) End() {
	if a.inner != nil {
		a.inner.EndEvent()
	}
}

type directComposition struct {
	compDevice *d3d.IDCompositionDevice
	compTarget *d3d.IDCompositionTarget
	compVisual *d3d.IDCompositionVisual
}

func releaseCOM[T any](p *T) {
	if p != nil {
		com.Unknown(p).Release()
	}
}

// NewDirectXRendererDevices takes references to devices' objects, casting the device to a DXGI
// device for DirectComposition unless it is disabled.
func NewDirectXRendererDevices(directxDevices *DirectXDevices, disableDirectComposition bool) (*DirectXRendererDevices, error) {
	var dxgiDevice *d3d.IDXGIDevice
	if !disableDirectComposition {
		var err error
		dxgiDevice, err = directxDevices.Device.CastDXGIDevice()
		if err != nil {
			return nil, fmt.Errorf("Creating DXGI device: %w", err)
		}
	}
	annotation, err := directxDevices.DeviceContext.CastUserDefinedAnnotation()
	if err != nil {
		annotation = nil
	}
	deviceContext1, err := directxDevices.DeviceContext.CastDeviceContext1()
	if err != nil {
		deviceContext1 = nil
	}
	return &DirectXRendererDevices{
		Adapter:        directxDevices.Adapter.Clone(),
		DxgiFactory:    directxDevices.DxgiFactory.Clone(),
		Device:         directxDevices.Device.Clone(),
		DeviceContext:  directxDevices.DeviceContext.Clone(),
		deviceContext1: deviceContext1,
		dxgiDevice:     dxgiDevice,
		annotation:     annotation,
	}, nil
}

// Release drops the references. It is idempotent.
func (d *DirectXRendererDevices) Release() {
	if d == nil || d.Device == nil {
		return
	}
	releaseCOM(d.annotation)
	releaseCOM(d.dxgiDevice)
	releaseCOM(d.deviceContext1)
	d.DeviceContext.Release()
	d.Device.Release()
	d.DxgiFactory.Release()
	d.Adapter.Release()
	*d = DirectXRendererDevices{}
}

// NewDirectXRenderer creates a renderer for hwnd on directxDevices with a 1x1 drawable.
func NewDirectXRenderer(hwnd win32.HWND, directxDevices *DirectXDevices, disableDirectComposition bool) (*DirectXRenderer, error) {
	if disableDirectComposition {
		slog.Info("Direct Composition is disabled.")
	}
	devices, err := NewDirectXRendererDevices(directxDevices, disableDirectComposition)
	if err != nil {
		return nil, fmt.Errorf("Creating DirectX devices: %w", err)
	}
	atlas := NewDirectXAtlas(devices.Device, devices.DeviceContext)
	fail := func(err error) (*DirectXRenderer, error) {
		atlas.Release()
		devices.Release()
		return nil, err
	}
	resources, err := newDirectXResources(devices, 1, 1, hwnd, disableDirectComposition)
	if err != nil {
		return fail(fmt.Errorf("Creating DirectX resources: %w", err))
	}
	globals, err := newDirectXGlobalElements(devices.Device)
	if err != nil {
		resources.release()
		return fail(fmt.Errorf("Creating DirectX global elements: %w", err))
	}
	pipelines, err := newDirectXRenderPipelines(devices.Device)
	if err != nil {
		globals.release()
		resources.release()
		return fail(fmt.Errorf("Creating DirectX render pipelines: %w", err))
	}
	var composition *directComposition
	if !disableDirectComposition {
		composition, err = newDirectComposition(devices.dxgiDevice, hwnd)
		if err == nil {
			if setErr := composition.setSwapChain(resources.swapChain); setErr != nil {
				composition.release()
				err = fmt.Errorf("Setting swap chain for DirectComposition: %w", setErr)
			}
		} else {
			err = fmt.Errorf("Creating DirectComposition: %w", err)
		}
		if err != nil {
			pipelines.release()
			globals.release()
			resources.release()
			return fail(err)
		}
	}
	return &DirectXRenderer{
		hwnd:              hwnd,
		atlas:             atlas,
		devices:           devices,
		resources:         resources,
		globals:           globals,
		pipelines:         pipelines,
		directComposition: composition,
		fontInfo:          directXRendererGetFontInfo(),
		width:             1,
		height:            1,
		skipDraws:         false,
		background:        gpui.WindowBackgroundAppearanceOpaque,
	}, nil
}

// SpriteAtlas returns the renderer's atlas.
func (r *DirectXRenderer) SpriteAtlas() gpui.PlatformAtlas {
	return r.atlas
}

func (r *DirectXRenderer) preDraw(clearColor *[4]float32, depthScale float32) error {
	if r.resources == nil {
		panic("resources missing")
	}
	if r.devices == nil {
		panic("devices missing")
	}
	resources := r.resources
	deviceContext := r.devices.DeviceContext
	isBgr := uint32(0)
	if r.fontInfo.isBgr {
		isBgr = 1
	}
	if err := updateBuffer(deviceContext, r.globals.globalParamsBuffer, []globalParams{{
		gammaRatios:               r.fontInfo.gammaRatios,
		viewportSize:              [2]float32{resources.viewport.Width, resources.viewport.Height},
		grayscaleEnhancedContrast: r.fontInfo.grayscaleEnhancedContrast,
		subpixelEnhancedContrast:  r.fontInfo.subpixelEnhancedContrast,
		isBgr:                     isBgr,
		depthScale:                depthScale,
	}}); err != nil {
		return err
	}
	if resources.renderTargetView == nil {
		return errors.New("missing render target view")
	}
	deviceContext.ClearRenderTargetView(resources.renderTargetView, clearColor)
	if resources.depthStencilView == nil {
		return errors.New("missing depth stencil view")
	}
	deviceContext.ClearDepthStencilView(resources.depthStencilView, d3d.D3D11_CLEAR_DEPTH, 0.0, 0)
	deviceContext.OMSetRenderTargets([]*d3d.ID3D11RenderTargetView{resources.renderTargetView}, resources.depthStencilView)
	deviceContext.RSSetViewports([]d3d.D3D11_VIEWPORT{resources.viewport})
	deviceContext.VSSetConstantBuffers(0, []*d3d.ID3D11Buffer{r.globals.globalParamsBuffer})
	deviceContext.VSSetConstantBuffers(1, []*d3d.ID3D11Buffer{r.globals.batchParamsBuffer})
	deviceContext.PSSetConstantBuffers(0, []*d3d.ID3D11Buffer{r.globals.globalParamsBuffer})
	return nil
}

func (r *DirectXRenderer) discardDepthBuffer() bool {
	if r.devices == nil || r.devices.deviceContext1 == nil {
		return false
	}
	if r.resources == nil || r.resources.depthStencilView == nil {
		return false
	}
	r.devices.deviceContext1.DiscardView(com.Unknown(r.resources.depthStencilView))
	return true
}

func (r *DirectXRenderer) present() error {
	if r.resources == nil {
		panic("resources missing")
	}
	if err := r.resources.swapChain.Present(0, 0).Err(); err != nil {
		return fmt.Errorf("Presenting swap chain failed: %w", err)
	}
	return nil
}

// HandleDeviceLost rebuilds every GPU object on directxDevices after the device was lost,
// retrying as upstream does; the next frame is skipped until MarkDrawable.
func (r *DirectXRenderer) HandleDeviceLost(directxDevices *DirectXDevices) error {
	_, err := tryToRecoverFromDeviceLost(func() (struct{}, error) {
		if err := r.handleDeviceLostImpl(directxDevices); err != nil {
			return struct{}{}, fmt.Errorf("DirectXRenderer handling device lost: %w", err)
		}
		return struct{}{}, nil
	})
	return err
}

func (r *DirectXRenderer) handleDeviceLostImpl(directxDevices *DirectXDevices) error {
	disableDirectComposition := r.directComposition == nil

	if gpuiutil.DebugAssertions && r.devices != nil {
		gpuiutil.LogErr(wrapErr("Failed to report live objects after device lost", reportLiveObjects(r.devices.Device)))
	}
	if r.resources != nil {
		r.resources.release()
		r.resources = nil
	}
	if r.devices != nil {
		r.devices.DeviceContext.OMSetRenderTargets(nil, nil)
		r.devices.DeviceContext.ClearState()
		r.devices.DeviceContext.Flush()
		if gpuiutil.DebugAssertions {
			gpuiutil.LogErr(wrapErr("Failed to report live objects after device lost", reportLiveObjects(r.devices.Device)))
		}
	}
	if r.directComposition != nil {
		r.directComposition.release()
		r.directComposition = nil
	}
	if r.devices != nil {
		r.devices.Release()
		r.devices = nil
	}

	devices, err := NewDirectXRendererDevices(directxDevices, disableDirectComposition)
	if err != nil {
		return fmt.Errorf("Recreating DirectX devices: %w", err)
	}
	resources, err := newDirectXResources(devices, r.width, r.height, r.hwnd, disableDirectComposition)
	if err != nil {
		devices.Release()
		return fmt.Errorf("Creating DirectX resources: %w", err)
	}
	globals, err := newDirectXGlobalElements(devices.Device)
	if err != nil {
		resources.release()
		devices.Release()
		return fmt.Errorf("Creating DirectXGlobalElements: %w", err)
	}
	pipelines, err := newDirectXRenderPipelines(devices.Device)
	if err != nil {
		globals.release()
		resources.release()
		devices.Release()
		return fmt.Errorf("Creating DirectXRenderPipelines: %w", err)
	}
	var composition *directComposition
	if !disableDirectComposition {
		composition, err = newDirectComposition(devices.dxgiDevice, r.hwnd)
		if err == nil {
			if err = composition.setSwapChain(resources.swapChain); err != nil {
				composition.release()
			}
		}
		if err != nil {
			pipelines.release()
			globals.release()
			resources.release()
			devices.Release()
			return err
		}
	}

	r.atlas.HandleDeviceLost(devices.Device, devices.DeviceContext)

	devices.DeviceContext.OMSetRenderTargets([]*d3d.ID3D11RenderTargetView{resources.renderTargetView}, resources.depthStencilView)
	r.devices = devices
	r.resources = resources
	r.globals.release()
	r.globals = globals
	r.pipelines.release()
	r.pipelines = pipelines
	r.directComposition = composition
	r.skipDraws = true
	return nil
}

func wrapErr(msg string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", msg, err)
}

// DrawWithAppearance renders scene with the clear color of backgroundAppearance and presents it;
// it draws nothing while a device-lost recovery is pending.
func (r *DirectXRenderer) DrawWithAppearance(scene *gpui.Scene, backgroundAppearance gpui.WindowBackgroundAppearance) error {
	if r.skipDraws {
		return nil
	}
	if err := r.render(scene, backgroundAppearance); err != nil {
		return err
	}
	return r.present()
}

// Draw renders scene with the background set by UpdateTransparency and presents it, logging a
// failure; it reports whether the frame was drawn.
func (r *DirectXRenderer) Draw(scene *gpui.Scene) bool {
	skipped := r.skipDraws
	if err := r.DrawWithAppearance(scene, r.background); err != nil {
		gpuiutil.LogErr(err)
		return false
	}
	return !skipped
}

func (r *DirectXRenderer) render(scene *gpui.Scene, backgroundAppearance gpui.WindowBackgroundAppearance) error {
	depthScale := gpui.QuadDepthScale(len(scene.Quads))
	clearColor := [4]float32{}
	if backgroundAppearance == gpui.WindowBackgroundAppearanceOpaque {
		clearColor = [4]float32{1, 1, 1, 1}
	}
	if err := r.preDraw(&clearColor, depthScale); err != nil {
		return err
	}
	if err := r.uploadSceneBuffers(scene); err != nil {
		return err
	}

	var ann *d3d.ID3DUserDefinedAnnotation
	if r.devices != nil && r.devices.annotation != nil && r.devices.annotation.GetStatus() {
		ann = r.devices.annotation
	}

	if len(scene.Quads) != 0 {
		var a annotation
		if ann != nil {
			a = newAnnotation(ann, fmt.Sprintf("opaque quads (%d)", len(scene.Quads)))
		}
		err := r.drawOpaqueQuads(len(scene.Quads))
		a.End()
		if err != nil {
			return err
		}
	}

	quadCursor := uint32(0)
	pathTileCursor := 0
	for batch := range scene.Batches() {
		var a annotation
		if ann != nil {
			a = newAnnotation(ann, batch.Label())
		}
		err := r.drawBatch(scene, batch, depthScale, &quadCursor, &pathTileCursor)
		a.End()
		if err != nil {
			return fmt.Errorf("scene too large:%d paths, %d shadows, %d quads, %d underlines, %d mono, %d subpixel, %d poly, %d surfaces: %w",
				len(scene.Paths),
				len(scene.Shadows),
				len(scene.Quads),
				len(scene.Underlines),
				len(scene.MonochromeSprites),
				len(scene.SubpixelSprites),
				len(scene.PolychromeSprites),
				len(scene.Surfaces),
				err,
			)
		}
	}
	r.discardDepthBuffer()
	return nil
}

func (r *DirectXRenderer) drawBatch(scene *gpui.Scene, batch gpui.PrimitiveBatch, depthScale float32, quadCursor *uint32, pathTileCursor *int) error {
	if _, ok := batch.(gpui.PrimitiveBatchQuads); ok {
		if err := r.setViewportDepthRange(0.0, 1.0); err != nil {
			return err
		}
	} else {
		depth := gpui.QuadDepth(*quadCursor, depthScale)
		if err := r.setViewportDepthRange(depth, depth); err != nil {
			return err
		}
	}
	switch b := batch.(type) {
	case gpui.PrimitiveBatchShadows:
		return r.drawShadows(b.Start, b.End-b.Start)
	case gpui.PrimitiveBatchQuads:
		*quadCursor += uint32(b.End - b.Start)
		return r.drawQuads(b.Start, b.End-b.Start)
	case gpui.PrimitiveBatchPaths:
		tilesStart := *pathTileCursor
		for _, path := range scene.Paths[b.Start:b.End] {
			*pathTileCursor += path.TileCount()
		}
		return r.drawPaths(tilesStart, *pathTileCursor)
	case gpui.PrimitiveBatchUnderlines:
		return r.drawUnderlines(b.Start, b.End-b.Start)
	case gpui.PrimitiveBatchMonochromeSprites:
		return r.drawMonochromeSprites(b.TextureID, b.Range.Start, b.Range.End-b.Range.Start)
	case gpui.PrimitiveBatchSubpixelSprites:
		return r.drawSubpixelSprites(b.TextureID, b.Range.Start, b.Range.End-b.Range.Start)
	case gpui.PrimitiveBatchPolychromeSprites:
		return r.drawPolychromeSprites(b.TextureID, b.Range.Start, b.Range.End-b.Range.Start)
	case gpui.PrimitiveBatchSurfaces:
		return r.drawSurfaces(scene.Surfaces[b.Start:b.End])
	case gpui.PrimitiveBatchNativeViews:
		return nil
	}
	return nil
}

// RenderToImage renders scene into the render target without presenting and reads it back as
// RGBA.
func (r *DirectXRenderer) RenderToImage(scene *gpui.Scene, backgroundAppearance gpui.WindowBackgroundAppearance) (*image.RGBA, error) {
	if r.skipDraws {
		return nil, errors.New("render_to_image unavailable while recovering from a lost device")
	}
	if err := r.render(scene, backgroundAppearance); err != nil {
		return nil, err
	}
	if r.devices == nil {
		return nil, errors.New("devices missing")
	}
	device := r.devices.Device
	context := r.devices.DeviceContext
	if r.resources == nil {
		return nil, errors.New("resources missing")
	}
	renderTarget := r.resources.renderTarget
	if renderTarget == nil {
		return nil, errors.New("render target missing")
	}

	desc := renderTarget.GetDesc()
	width := desc.Width
	height := desc.Height
	stagingDesc := desc
	stagingDesc.Usage = d3d.D3D11_USAGE_STAGING
	stagingDesc.BindFlags = 0
	stagingDesc.CPUAccessFlags = d3d.D3D11_CPU_ACCESS_READ
	stagingDesc.MiscFlags = 0
	stagingDesc.MipLevels = 1
	stagingDesc.ArraySize = 1
	stagingDesc.SampleDesc = d3d.DXGI_SAMPLE_DESC{Count: 1, Quality: 0}
	staging, err := device.CreateTexture2D(&stagingDesc, nil)
	if err != nil {
		return nil, err
	}
	if staging == nil {
		return nil, errors.New("creating staging texture")
	}
	defer staging.Release()
	context.CopyResource(staging.Resource(), renderTarget.Resource())

	mapped, err := context.Map(staging.Resource(), 0, d3d.D3D11_MAP_READ, 0)
	if err != nil {
		return nil, err
	}
	rowBytes := int(width) * 4
	pixels := make([]byte, rowBytes*int(height))
	src := unsafe.Slice((*byte)(mapped.PData), int(mapped.RowPitch)*int(height))
	for row := range int(height) {
		s := row * int(mapped.RowPitch)
		copy(pixels[row*rowBytes:(row+1)*rowBytes], src[s:s+rowBytes])
	}
	context.Unmap(staging.Resource(), 0)
	for i := 0; i+3 < len(pixels); i += 4 {
		pixels[i], pixels[i+2] = pixels[i+2], pixels[i]
	}
	return &image.RGBA{Pix: pixels, Stride: rowBytes, Rect: image.Rect(0, 0, int(width), int(height))}, nil
}

func (r *DirectXRenderer) setViewportDepthRange(minimumDepth, maximumDepth float32) error {
	if r.devices == nil {
		return errors.New("devices missing")
	}
	if r.resources == nil {
		return errors.New("resources missing")
	}
	viewport := r.resources.viewport
	viewport.MinDepth = minimumDepth
	viewport.MaxDepth = maximumDepth
	r.devices.DeviceContext.RSSetViewports([]d3d.D3D11_VIEWPORT{viewport})
	return nil
}

func (r *DirectXRenderer) batchParamsBuffer() (*d3d.ID3D11Buffer, error) {
	if r.globals.batchParamsBuffer == nil {
		return nil, errors.New("batch params buffer missing")
	}
	return r.globals.batchParamsBuffer, nil
}

func (r *DirectXRenderer) drawOpaqueQuads(quadCount int) error {
	if r.devices == nil {
		return errors.New("devices missing")
	}
	buffer, err := r.batchParamsBuffer()
	if err != nil {
		return err
	}
	return r.pipelines.opaqueQuadPipeline.draw(r.devices.DeviceContext, r.pipelines.quadPipeline, buffer, uint32(quadCount))
}

// Resize resizes the swap chain and recreates the render target and depth buffer; sizes are
// clamped to at least 1x1.
func (r *DirectXRenderer) Resize(newSize gpui.Size[gpui.DevicePixels]) error {
	width := uint32(max(int32(newSize.Width), 1))
	height := uint32(max(int32(newSize.Height), 1))
	if r.width == width && r.height == height {
		return nil
	}
	r.width = width
	r.height = height

	if r.devices == nil {
		return errors.New("devices missing")
	}
	devices := r.devices
	devices.DeviceContext.OMSetRenderTargets(nil, nil)
	if r.resources == nil {
		return errors.New("resources missing")
	}
	resources := r.resources
	releaseCOM(resources.renderTarget)
	resources.renderTarget = nil
	releaseCOM(resources.renderTargetView)
	resources.renderTargetView = nil

	if err := resources.swapChain.ResizeBuffers(bufferCount, width, height, renderTargetFormat, 0); err != nil {
		return fmt.Errorf("Failed to resize swap chain: %w", err)
	}
	if err := resources.recreateResources(devices, width, height); err != nil {
		return err
	}
	devices.DeviceContext.OMSetRenderTargets([]*d3d.ID3D11RenderTargetView{resources.renderTargetView}, resources.depthStencilView)
	return nil
}

// UpdateDrawableSize resizes the drawable, logging a failure.
func (r *DirectXRenderer) UpdateDrawableSize(size gpui.Size[gpui.DevicePixels]) {
	gpuiutil.LogErr(r.Resize(size))
}

// UpdateTransparency selects the background Draw clears to.
func (r *DirectXRenderer) UpdateTransparency(transparent bool) {
	if transparent {
		r.background = gpui.WindowBackgroundAppearanceTransparent
	} else {
		r.background = gpui.WindowBackgroundAppearanceOpaque
	}
}

// SetSubpixelLayout does nothing: the subpixel order comes from the system's DirectWrite
// rendering parameters.
func (r *DirectXRenderer) SetSubpixelLayout(isBGR bool) {}

// SupportsDualSourceBlending reports true: the subpixel pipeline blends with SRC1 on every
// Direct3D 11 device.
func (r *DirectXRenderer) SupportsDualSourceBlending() bool { return true }

// DeviceLost reports false: the platform detects a lost device and calls HandleDeviceLost.
func (r *DirectXRenderer) DeviceLost() bool { return false }

// Recover does nothing: recovery goes through HandleDeviceLost with the platform's new devices.
func (r *DirectXRenderer) Recover(handle gpui.RawWindowHandle) error { return nil }

func (r *DirectXRenderer) uploadSceneBuffers(scene *gpui.Scene) error {
	if r.devices == nil {
		return errors.New("devices missing")
	}
	device := r.devices.Device
	context := r.devices.DeviceContext
	p := r.pipelines

	if len(scene.Shadows) != 0 {
		if err := p.shadowPipeline.updateBuffer(device, context, scene.Shadows); err != nil {
			return err
		}
	}
	if len(scene.Quads) != 0 {
		if err := p.quadPipeline.updateBuffer(device, context, scene.Quads); err != nil {
			return err
		}
	}
	if len(scene.Paths) != 0 {
		paints := make([]gpui.PathPaint, 0, len(scene.Paths))
		var curves []gpui.PathCurve
		var tiles []gpui.PathTile
		for _, path := range scene.Paths {
			path.AppendTo(&paints, &curves, &tiles)
		}
		if err := p.pathTilePipeline.updateBuffer(device, context, tiles); err != nil {
			return err
		}
		if err := p.pathCurves.write(device, context, curves); err != nil {
			return err
		}
		if err := p.pathPaints.write(device, context, paints); err != nil {
			return err
		}
		buffers := []*d3d.ID3D11ShaderResourceView{p.pathCurves.view, p.pathPaints.view}
		context.VSSetShaderResources(2, buffers)
		context.PSSetShaderResources(2, buffers)
	}
	if len(scene.Underlines) != 0 {
		if err := p.underlinePipeline.updateBuffer(device, context, scene.Underlines); err != nil {
			return err
		}
	}
	if len(scene.MonochromeSprites) != 0 {
		if err := p.monoSprites.updateBuffer(device, context, scene.MonochromeSprites); err != nil {
			return err
		}
	}
	if len(scene.SubpixelSprites) != 0 {
		if err := p.subpixelSprites.updateBuffer(device, context, scene.SubpixelSprites); err != nil {
			return err
		}
	}
	if len(scene.PolychromeSprites) != 0 {
		if err := p.polySprites.updateBuffer(device, context, scene.PolychromeSprites); err != nil {
			return err
		}
	}
	return nil
}

func drawRangeOf[T any](r *DirectXRenderer, pipeline *pipelineState[T], start, length int) error {
	if length == 0 {
		return nil
	}
	if r.devices == nil {
		return errors.New("devices missing")
	}
	buffer, err := r.batchParamsBuffer()
	if err != nil {
		return err
	}
	return pipeline.drawRange(r.devices.DeviceContext, buffer, uint32(start), uint32(length))
}

func (r *DirectXRenderer) drawShadows(start, length int) error {
	return drawRangeOf(r, r.pipelines.shadowPipeline, start, length)
}

func (r *DirectXRenderer) drawQuads(start, length int) error {
	return drawRangeOf(r, r.pipelines.quadPipeline, start, length)
}

func (r *DirectXRenderer) drawPaths(tileStart, tileEnd int) error {
	if tileStart >= tileEnd {
		return nil
	}
	return drawRangeOf(r, r.pipelines.pathTilePipeline, tileStart, tileEnd-tileStart)
}

func (r *DirectXRenderer) drawUnderlines(start, length int) error {
	return drawRangeOf(r, r.pipelines.underlinePipeline, start, length)
}

func drawSpritesOf[T any](r *DirectXRenderer, pipeline *pipelineState[T], textureID gpui.AtlasTextureID, start, length int) error {
	if length == 0 {
		return nil
	}
	if r.devices == nil {
		return errors.New("devices missing")
	}
	textureView, ok := r.atlas.GetTextureView(textureID)
	if !ok {
		return nil
	}
	defer releaseCOM(textureView[0])
	buffer, err := r.batchParamsBuffer()
	if err != nil {
		return err
	}
	return pipeline.drawRangeWithTexture(r.devices.DeviceContext, textureView[:], buffer, []*d3d.ID3D11SamplerState{r.globals.sampler}, uint32(start), uint32(length))
}

func (r *DirectXRenderer) drawMonochromeSprites(textureID gpui.AtlasTextureID, start, length int) error {
	return drawSpritesOf(r, r.pipelines.monoSprites, textureID, start, length)
}

func (r *DirectXRenderer) drawSubpixelSprites(textureID gpui.AtlasTextureID, start, length int) error {
	return drawSpritesOf(r, r.pipelines.subpixelSprites, textureID, start, length)
}

func (r *DirectXRenderer) drawPolychromeSprites(textureID gpui.AtlasTextureID, start, length int) error {
	return drawSpritesOf(r, r.pipelines.polySprites, textureID, start, length)
}

func (r *DirectXRenderer) drawSurfaces(surfaces []gpui.PaintSurface) error {
	if len(surfaces) == 0 {
		return nil
	}
	return nil
}

// GPUSpecs describes the adapter, logging a failure.
func (r *DirectXRenderer) GPUSpecs() (gpui.GPUSpecs, bool) {
	return gpuiutil.Result(r.gpuSpecs()).LogErr()
}

func (r *DirectXRenderer) gpuSpecs() (gpui.GPUSpecs, error) {
	if r.devices == nil {
		return gpui.GPUSpecs{}, errors.New("devices missing")
	}
	desc, err := r.devices.Adapter.GetDesc1()
	if err != nil {
		return gpui.GPUSpecs{}, err
	}
	isSoftwareEmulated := desc.Flags&d3d.DXGI_ADAPTER_FLAG_SOFTWARE != 0
	deviceName := desc.DescriptionString()
	var driverName string
	switch desc.VendorId {
	case 0x10DE:
		driverName = "NVIDIA Corporation"
	case 0x1002:
		driverName = "AMD Corporation"
	case 0x8086:
		driverName = "Intel Corporation"
	default:
		driverName = fmt.Sprintf("Unknown Vendor (ID: %#X)", desc.VendorId)
	}
	var driverVersion string
	switch desc.VendorId {
	case 0x10DE:
		driverVersion, err = nvidiaGetDriverVersion()
	case 0x1002:
		driverVersion, err = amdGetDriverVersion()
	default:
		driverVersion, err = dxgiGetDriverVersion(r.devices.Adapter)
	}
	if err != nil {
		gpuiutil.LogErr(fmt.Errorf("Failed to get gpu driver info: %w", err))
		driverVersion = "Unknown Driver"
	}
	return gpui.GPUSpecs{
		IsSoftwareEmulated: isSoftwareEmulated,
		DeviceName:         deviceName,
		DriverName:         driverName,
		DriverInfo:         driverVersion,
	}, nil
}

var cachedFontInfo = sync.OnceValue(func() *directXFontInfo {
	factory, err := dwrite.DWriteCreateFactory(dwrite.DWRITE_FACTORY_TYPE_SHARED)
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer factory.Release()
	base, err := factory.CreateRenderingParams()
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer base.Release()
	renderParams, err := base.CastRenderingParams1()
	if err != nil {
		panic("called `Result::unwrap()` on an `Err` value: " + err.Error())
	}
	defer renderParams.Release()
	return &directXFontInfo{
		gammaRatios:               gpui.GetGammaCorrectionRatios(renderParams.GetGamma()),
		grayscaleEnhancedContrast: renderParams.GetGrayscaleEnhancedContrast(),
		subpixelEnhancedContrast:  renderParams.GetEnhancedContrast(),
		isBgr:                     renderParams.GetPixelGeometry() == dwrite.DWRITE_PIXEL_GEOMETRY_BGR,
	}
})

func directXRendererGetFontInfo() *directXFontInfo {
	return cachedFontInfo()
}

// MarkDrawable ends the frame skipping that follows a device-lost recovery.
func (r *DirectXRenderer) MarkDrawable() {
	r.skipDraws = false
}

// Destroy releases every GPU object, the atlas included, as dropping the Rust renderer does; in
// debug builds it first reports live device objects. It is idempotent.
func (r *DirectXRenderer) Destroy() {
	if gpuiutil.DebugAssertions && r.devices != nil {
		_ = reportLiveObjects(r.devices.Device)
	}
	if r.pipelines != nil {
		r.pipelines.release()
		r.pipelines = nil
	}
	if r.globals != nil {
		r.globals.release()
		r.globals = nil
	}
	if r.directComposition != nil {
		r.directComposition.release()
		r.directComposition = nil
	}
	if r.resources != nil {
		r.resources.release()
		r.resources = nil
	}
	if r.atlas != nil {
		r.atlas.Release()
	}
	if r.devices != nil {
		r.devices.Release()
		r.devices = nil
	}
}

func newDirectXResources(devices *DirectXRendererDevices, width, height uint32, hwnd win32.HWND, disableDirectComposition bool) (*directXResources, error) {
	var swapChain *d3d.IDXGISwapChain1
	var err error
	if disableDirectComposition {
		swapChain, err = createSwapChain(devices.DxgiFactory, devices.Device, hwnd, width, height)
	} else {
		swapChain, err = createSwapChainForComposition(devices.DxgiFactory, devices.Device, width, height)
	}
	if err != nil {
		return nil, err
	}
	res := &directXResources{swapChain: swapChain}
	if err := res.recreateResources(devices, width, height); err != nil {
		swapChain.Release()
		return nil, err
	}
	if err := setRasterizerState(devices.Device, devices.DeviceContext); err != nil {
		res.release()
		return nil, err
	}
	return res, nil
}

func (res *directXResources) recreateResources(devices *DirectXRendererDevices, width, height uint32) error {
	renderTarget, renderTargetView, depthStencilTexture, depthStencilView, viewport, err := createResources(devices, res.swapChain, width, height)
	if err != nil {
		return err
	}
	releaseCOM(res.renderTarget)
	releaseCOM(res.renderTargetView)
	releaseCOM(res.depthStencilTexture)
	releaseCOM(res.depthStencilView)
	res.renderTarget = renderTarget
	res.renderTargetView = renderTargetView
	res.depthStencilTexture = depthStencilTexture
	res.depthStencilView = depthStencilView
	res.viewport = viewport
	return nil
}

func (res *directXResources) release() {
	releaseCOM(res.depthStencilView)
	releaseCOM(res.depthStencilTexture)
	releaseCOM(res.renderTargetView)
	releaseCOM(res.renderTarget)
	releaseCOM(res.swapChain)
	*res = directXResources{}
}

func newDirectXRenderPipelines(device *d3d.ID3D11Device) (p *directXRenderPipelines, err error) {
	p = &directXRenderPipelines{}
	defer func() {
		if err != nil {
			p.release()
			p = nil
		}
	}()
	blend := func() (*d3d.ID3D11BlendState, error) { return createBlendState(device) }
	if p.shadowPipeline, err = newPipelineStateWith[gpui.Shadow](device, "shadow_pipeline", ShaderModuleShadow, 4, blend); err != nil {
		return
	}
	if p.quadPipeline, err = newPipelineStateWith[gpui.Quad](device, "quad_pipeline", ShaderModuleQuad, 64, blend); err != nil {
		return
	}
	if p.pathTilePipeline, err = newPipelineStateWith[gpui.PathTile](device, "path_tile_pipeline", ShaderModulePathTile, 64, blend); err != nil {
		return
	}
	if p.pathCurves, err = newStructuredBuffer[gpui.PathCurve](device, "path_curves", 256); err != nil {
		return
	}
	if p.pathPaints, err = newStructuredBuffer[gpui.PathPaint](device, "path_paints", 64); err != nil {
		return
	}
	if p.underlinePipeline, err = newPipelineStateWith[gpui.Underline](device, "underline_pipeline", ShaderModuleUnderline, 4, blend); err != nil {
		return
	}
	if p.monoSprites, err = newPipelineStateWith[gpui.MonochromeSprite](device, "monochrome_sprite_pipeline", ShaderModuleMonochromeSprite, 512, blend); err != nil {
		return
	}
	subpixelBlend := func() (*d3d.ID3D11BlendState, error) { return createBlendStateForSubpixelRendering(device) }
	if p.subpixelSprites, err = newPipelineStateWith[gpui.SubpixelSprite](device, "subpixel_sprite_pipeline", ShaderModuleSubpixelSprite, 512, subpixelBlend); err != nil {
		return
	}
	if p.polySprites, err = newPipelineStateWith[gpui.PolychromeSprite](device, "polychrome_sprite_pipeline", ShaderModulePolychromeSprite, 16, blend); err != nil {
		return
	}
	p.opaqueQuadPipeline, err = newOpaqueQuadPipeline(device)
	return
}

func (p *directXRenderPipelines) release() {
	if p == nil {
		return
	}
	p.shadowPipeline.release()
	p.quadPipeline.release()
	p.opaqueQuadPipeline.release()
	p.pathTilePipeline.release()
	p.pathCurves.release()
	p.pathPaints.release()
	p.underlinePipeline.release()
	p.monoSprites.release()
	p.subpixelSprites.release()
	p.polySprites.release()
	*p = directXRenderPipelines{}
}

func newDirectComposition(dxgiDevice *d3d.IDXGIDevice, hwnd win32.HWND) (*directComposition, error) {
	compDevice, err := getCompDevice(dxgiDevice)
	if err != nil {
		return nil, err
	}
	compTarget, err := compDevice.CreateTargetForHwnd(hwnd, false)
	if err != nil {
		compDevice.Release()
		return nil, err
	}
	compVisual, err := compDevice.CreateVisual()
	if err != nil {
		compTarget.Release()
		compDevice.Release()
		return nil, err
	}
	return &directComposition{compDevice: compDevice, compTarget: compTarget, compVisual: compVisual}, nil
}

func (c *directComposition) setSwapChain(swapChain *d3d.IDXGISwapChain1) error {
	if err := c.compVisual.SetContent(com.Unknown(swapChain)); err != nil {
		return err
	}
	if err := c.compTarget.SetRoot(c.compVisual); err != nil {
		return err
	}
	return c.compDevice.Commit()
}

func (c *directComposition) release() {
	releaseCOM(c.compVisual)
	releaseCOM(c.compTarget)
	releaseCOM(c.compDevice)
	*c = directComposition{}
}

func newDirectXGlobalElements(device *d3d.ID3D11Device) (*directXGlobalElements, error) {
	globalParamsBuffer, err := createConstantBuffer[globalParams](device)
	if err != nil {
		return nil, err
	}
	batchParamsBuffer, err := createConstantBuffer[batchParams](device)
	if err != nil {
		releaseCOM(globalParamsBuffer)
		return nil, err
	}
	desc := d3d.D3D11_SAMPLER_DESC{
		Filter:         d3d.D3D11_FILTER_MIN_MAG_MIP_LINEAR,
		AddressU:       d3d.D3D11_TEXTURE_ADDRESS_WRAP,
		AddressV:       d3d.D3D11_TEXTURE_ADDRESS_WRAP,
		AddressW:       d3d.D3D11_TEXTURE_ADDRESS_WRAP,
		MipLODBias:     0.0,
		MaxAnisotropy:  1,
		ComparisonFunc: d3d.D3D11_COMPARISON_ALWAYS,
		BorderColor:    [4]float32{},
		MinLOD:         0.0,
		MaxLOD:         d3d.D3D11_FLOAT32_MAX,
	}
	sampler, err := device.CreateSamplerState(&desc)
	if err != nil {
		releaseCOM(batchParamsBuffer)
		releaseCOM(globalParamsBuffer)
		return nil, err
	}
	return &directXGlobalElements{
		globalParamsBuffer: globalParamsBuffer,
		batchParamsBuffer:  batchParamsBuffer,
		sampler:            sampler,
	}, nil
}

func (g *directXGlobalElements) release() {
	if g == nil {
		return
	}
	releaseCOM(g.sampler)
	releaseCOM(g.batchParamsBuffer)
	releaseCOM(g.globalParamsBuffer)
	*g = directXGlobalElements{}
}

type globalParams struct {
	gammaRatios               [4]float32
	viewportSize              [2]float32
	grayscaleEnhancedContrast float32
	subpixelEnhancedContrast  float32
	isBgr                     uint32
	depthScale                float32
	_                         [2]uint32
}

type batchParams struct {
	startIndex uint32
	_          [3]uint32
}

type structuredBuffer[T any] struct {
	label    string
	buffer   *d3d.ID3D11Buffer
	capacity int
	view     *d3d.ID3D11ShaderResourceView
}

func sizeOf[T any]() int {
	var zero T
	return int(unsafe.Sizeof(zero))
}

func newStructuredBuffer[T any](device *d3d.ID3D11Device, label string, capacity int) (*structuredBuffer[T], error) {
	buffer, err := createBuffer(device, sizeOf[T](), capacity)
	if err != nil {
		return nil, err
	}
	view, err := createBufferView(device, buffer)
	if err != nil {
		buffer.Release()
		return nil, err
	}
	return &structuredBuffer[T]{label: label, buffer: buffer, capacity: capacity, view: view}, nil
}

func (b *structuredBuffer[T]) write(device *d3d.ID3D11Device, deviceContext *d3d.ID3D11DeviceContext, data []T) error {
	if err := b.ensureCapacity(device, len(data)); err != nil {
		return err
	}
	return updateBuffer(deviceContext, b.buffer, data)
}

func (b *structuredBuffer[T]) ensureCapacity(device *d3d.ID3D11Device, length int) error {
	if b.capacity < length {
		elementSize := max(sizeOf[T](), 1)
		hi, lo := bits.Mul64(uint64(length), uint64(elementSize))
		if hi != 0 || lo > maxInstanceBufferSize {
			return fmt.Errorf("%s buffer needs %d elements of %d bytes, above the maximum of %d bytes", b.label, length, elementSize, maxInstanceBufferSize)
		}
		newCapacity := min(nextPowerOfTwo(length), maxInstanceBufferSize/elementSize)
		slog.Debug(fmt.Sprintf("Updating %s buffer size from %d to %d", b.label, b.capacity, newCapacity))
		buffer, err := createBuffer(device, sizeOf[T](), newCapacity)
		if err != nil {
			return err
		}
		view, err := createBufferView(device, buffer)
		if err != nil {
			buffer.Release()
			return err
		}
		releaseCOM(b.view)
		releaseCOM(b.buffer)
		b.buffer = buffer
		b.view = view
		b.capacity = newCapacity
	}
	return nil
}

func (b *structuredBuffer[T]) release() {
	if b == nil {
		return
	}
	releaseCOM(b.view)
	releaseCOM(b.buffer)
	b.view = nil
	b.buffer = nil
}

func nextPowerOfTwo(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << bits.Len(uint(n-1))
}

type pipelineState[T any] struct {
	vertex            *d3d.ID3D11VertexShader
	fragment          *d3d.ID3D11PixelShader
	instances         *structuredBuffer[T]
	blendState        *d3d.ID3D11BlendState
	depthStencilState *d3d.ID3D11DepthStencilState
}

func newPipelineStateWith[T any](device *d3d.ID3D11Device, label string, shaderModule ShaderModule, bufferSize int, blend func() (*d3d.ID3D11BlendState, error)) (*pipelineState[T], error) {
	blendState, err := blend()
	if err != nil {
		return nil, err
	}
	p, err := newPipelineState[T](device, label, shaderModule, bufferSize, blendState)
	if err != nil {
		blendState.Release()
		return nil, err
	}
	return p, nil
}

func newPipelineState[T any](device *d3d.ID3D11Device, label string, shaderModule ShaderModule, bufferSize int, blendState *d3d.ID3D11BlendState) (*pipelineState[T], error) {
	vertex, fragment, err := createShaders(device, shaderModule)
	if err != nil {
		return nil, err
	}
	instances, err := newStructuredBuffer[T](device, label, bufferSize)
	if err != nil {
		vertex.Release()
		fragment.Release()
		return nil, err
	}
	depthStencilState, err := createDepthStencilState(device, d3d.D3D11_DEPTH_WRITE_MASK_ZERO)
	if err != nil {
		instances.release()
		vertex.Release()
		fragment.Release()
		return nil, err
	}
	return &pipelineState[T]{
		vertex:            vertex,
		fragment:          fragment,
		instances:         instances,
		blendState:        blendState,
		depthStencilState: depthStencilState,
	}, nil
}

func createShaders(device *d3d.ID3D11Device, shaderModule ShaderModule) (*d3d.ID3D11VertexShader, *d3d.ID3D11PixelShader, error) {
	rawVertex, err := NewRawShaderBytes(shaderModule, ShaderTargetVertex)
	if err != nil {
		return nil, nil, err
	}
	vertex, err := createVertexShader(device, rawVertex.AsBytes())
	if err != nil {
		return nil, nil, err
	}
	rawFragment, err := NewRawShaderBytes(shaderModule, ShaderTargetFragment)
	if err != nil {
		vertex.Release()
		return nil, nil, err
	}
	fragment, err := createFragmentShader(device, rawFragment.AsBytes())
	if err != nil {
		vertex.Release()
		return nil, nil, err
	}
	return vertex, fragment, nil
}

func (p *pipelineState[T]) updateBuffer(device *d3d.ID3D11Device, deviceContext *d3d.ID3D11DeviceContext, data []T) error {
	return p.instances.write(device, deviceContext, data)
}

func (p *pipelineState[T]) drawRange(deviceContext *d3d.ID3D11DeviceContext, batchParamsBuffer *d3d.ID3D11Buffer, firstInstance, instanceCount uint32) error {
	if err := p.ensureRange(firstInstance, instanceCount); err != nil {
		return err
	}
	if err := updateBatchStart(deviceContext, batchParamsBuffer, firstInstance); err != nil {
		return err
	}
	setPipelineState(deviceContext, []*d3d.ID3D11ShaderResourceView{p.instances.view}, d3d.D3D_PRIMITIVE_TOPOLOGY_TRIANGLESTRIP, p.vertex, p.fragment, p.blendState, p.depthStencilState)
	deviceContext.DrawInstanced(4, instanceCount, 0, 0)
	return nil
}

func (p *pipelineState[T]) drawRangeWithTexture(deviceContext *d3d.ID3D11DeviceContext, texture []*d3d.ID3D11ShaderResourceView, batchParamsBuffer *d3d.ID3D11Buffer, sampler []*d3d.ID3D11SamplerState, firstInstance, instanceCount uint32) error {
	if err := p.ensureRange(firstInstance, instanceCount); err != nil {
		return err
	}
	if err := updateBatchStart(deviceContext, batchParamsBuffer, firstInstance); err != nil {
		return err
	}
	setPipelineState(deviceContext, []*d3d.ID3D11ShaderResourceView{p.instances.view}, d3d.D3D_PRIMITIVE_TOPOLOGY_TRIANGLESTRIP, p.vertex, p.fragment, p.blendState, p.depthStencilState)
	deviceContext.PSSetSamplers(0, sampler)
	deviceContext.VSSetShaderResources(0, texture)
	deviceContext.PSSetShaderResources(0, texture)
	deviceContext.DrawInstanced(4, instanceCount, 0, 0)
	return nil
}

func (p *pipelineState[T]) ensureRange(firstInstance, instanceCount uint32) error {
	if int(firstInstance)+int(instanceCount) > p.instances.capacity {
		return fmt.Errorf("DirectX instance range exceeds the %s buffer", p.instances.label)
	}
	return nil
}

func (p *pipelineState[T]) release() {
	if p == nil {
		return
	}
	releaseCOM(p.vertex)
	releaseCOM(p.fragment)
	p.instances.release()
	releaseCOM(p.blendState)
	releaseCOM(p.depthStencilState)
	*p = pipelineState[T]{}
}

type opaqueQuadPipeline struct {
	vertex            *d3d.ID3D11VertexShader
	fragment          *d3d.ID3D11PixelShader
	blendState        *d3d.ID3D11BlendState
	depthStencilState *d3d.ID3D11DepthStencilState
}

func newOpaqueQuadPipeline(device *d3d.ID3D11Device) (*opaqueQuadPipeline, error) {
	vertex, fragment, err := createShaders(device, ShaderModuleOpaqueQuad)
	if err != nil {
		return nil, err
	}
	blendState, err := createOpaqueBlendState(device)
	if err != nil {
		vertex.Release()
		fragment.Release()
		return nil, err
	}
	depthStencilState, err := createDepthStencilState(device, d3d.D3D11_DEPTH_WRITE_MASK_ALL)
	if err != nil {
		blendState.Release()
		vertex.Release()
		fragment.Release()
		return nil, err
	}
	return &opaqueQuadPipeline{vertex: vertex, fragment: fragment, blendState: blendState, depthStencilState: depthStencilState}, nil
}

func (o *opaqueQuadPipeline) draw(deviceContext *d3d.ID3D11DeviceContext, quadPipeline *pipelineState[gpui.Quad], batchParamsBuffer *d3d.ID3D11Buffer, quadCount uint32) error {
	if quadCount == 0 {
		return nil
	}
	if err := quadPipeline.ensureRange(0, quadCount); err != nil {
		return err
	}
	if err := updateBatchStart(deviceContext, batchParamsBuffer, quadCount-1); err != nil {
		return err
	}
	setPipelineState(deviceContext, []*d3d.ID3D11ShaderResourceView{quadPipeline.instances.view}, d3d.D3D_PRIMITIVE_TOPOLOGY_TRIANGLESTRIP, o.vertex, o.fragment, o.blendState, o.depthStencilState)
	deviceContext.DrawInstanced(4, quadCount, 0, 0)
	return nil
}

func (o *opaqueQuadPipeline) release() {
	if o == nil {
		return
	}
	releaseCOM(o.vertex)
	releaseCOM(o.fragment)
	releaseCOM(o.blendState)
	releaseCOM(o.depthStencilState)
	*o = opaqueQuadPipeline{}
}

func getCompDevice(dxgiDevice *d3d.IDXGIDevice) (*d3d.IDCompositionDevice, error) {
	return d3d.DCompositionCreateDevice(dxgiDevice)
}

func createSwapChainForComposition(dxgiFactory *d3d.IDXGIFactory6, device *d3d.ID3D11Device, width, height uint32) (*d3d.IDXGISwapChain1, error) {
	desc := d3d.DXGI_SWAP_CHAIN_DESC1{
		Width:       width,
		Height:      height,
		Format:      renderTargetFormat,
		Stereo:      d3d.BoolOf(false),
		SampleDesc:  d3d.DXGI_SAMPLE_DESC{Count: 1, Quality: 0},
		BufferUsage: d3d.DXGI_USAGE_RENDER_TARGET_OUTPUT,
		BufferCount: bufferCount,
		Scaling:     d3d.DXGI_SCALING_STRETCH,
		SwapEffect:  d3d.DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL,
		AlphaMode:   d3d.DXGI_ALPHA_MODE_PREMULTIPLIED,
		Flags:       0,
	}
	return dxgiFactory.CreateSwapChainForComposition(com.Unknown(device), &desc)
}

func createSwapChain(dxgiFactory *d3d.IDXGIFactory6, device *d3d.ID3D11Device, hwnd win32.HWND, width, height uint32) (*d3d.IDXGISwapChain1, error) {
	desc := d3d.DXGI_SWAP_CHAIN_DESC1{
		Width:       width,
		Height:      height,
		Format:      renderTargetFormat,
		Stereo:      d3d.BoolOf(false),
		SampleDesc:  d3d.DXGI_SAMPLE_DESC{Count: 1, Quality: 0},
		BufferUsage: d3d.DXGI_USAGE_RENDER_TARGET_OUTPUT,
		BufferCount: bufferCount,
		Scaling:     d3d.DXGI_SCALING_NONE,
		SwapEffect:  d3d.DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL,
		AlphaMode:   d3d.DXGI_ALPHA_MODE_IGNORE,
		Flags:       0,
	}
	swapChain, err := dxgiFactory.CreateSwapChainForHwnd(com.Unknown(device), hwnd, &desc)
	if err != nil {
		return nil, err
	}
	if err := dxgiFactory.MakeWindowAssociation(hwnd, d3d.DXGI_MWA_NO_ALT_ENTER); err != nil {
		swapChain.Release()
		return nil, err
	}
	return swapChain, nil
}

func createResources(devices *DirectXRendererDevices, swapChain *d3d.IDXGISwapChain1, width, height uint32) (*d3d.ID3D11Texture2D, *d3d.ID3D11RenderTargetView, *d3d.ID3D11Texture2D, *d3d.ID3D11DepthStencilView, d3d.D3D11_VIEWPORT, error) {
	renderTarget, renderTargetView, err := createRenderTargetAndItsView(swapChain, devices.Device)
	if err != nil {
		return nil, nil, nil, nil, d3d.D3D11_VIEWPORT{}, err
	}
	depthStencilTexture, depthStencilView, err := createDepthStencilTextureAndView(devices.Device, width, height)
	if err != nil {
		releaseCOM(renderTargetView)
		renderTarget.Release()
		return nil, nil, nil, nil, d3d.D3D11_VIEWPORT{}, err
	}
	viewport := setViewport(devices.DeviceContext, float32(width), float32(height))
	return renderTarget, renderTargetView, depthStencilTexture, depthStencilView, viewport, nil
}

func createRenderTargetAndItsView(swapChain *d3d.IDXGISwapChain1, device *d3d.ID3D11Device) (*d3d.ID3D11Texture2D, *d3d.ID3D11RenderTargetView, error) {
	renderTarget, err := swapChain.GetBuffer(0)
	if err != nil {
		return nil, nil, err
	}
	renderTargetView, err := device.CreateRenderTargetView(renderTarget.Resource())
	if err != nil {
		renderTarget.Release()
		return nil, nil, err
	}
	return renderTarget, renderTargetView, nil
}

func setViewport(deviceContext *d3d.ID3D11DeviceContext, width, height float32) d3d.D3D11_VIEWPORT {
	viewport := []d3d.D3D11_VIEWPORT{{
		TopLeftX: 0.0,
		TopLeftY: 0.0,
		Width:    width,
		Height:   height,
		MinDepth: 0.0,
		MaxDepth: 1.0,
	}}
	deviceContext.RSSetViewports(viewport)
	return viewport[0]
}

func createDepthStencilTextureAndView(device *d3d.ID3D11Device, width, height uint32) (*d3d.ID3D11Texture2D, *d3d.ID3D11DepthStencilView, error) {
	desc := d3d.D3D11_TEXTURE2D_DESC{
		Width:          width,
		Height:         height,
		MipLevels:      1,
		ArraySize:      1,
		Format:         d3d.DXGI_FORMAT_D32_FLOAT,
		SampleDesc:     d3d.DXGI_SAMPLE_DESC{Count: 1, Quality: 0},
		Usage:          d3d.D3D11_USAGE_DEFAULT,
		BindFlags:      d3d.D3D11_BIND_DEPTH_STENCIL,
		CPUAccessFlags: 0,
		MiscFlags:      0,
	}
	texture, err := device.CreateTexture2D(&desc, nil)
	if err != nil {
		return nil, nil, err
	}
	if texture == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	view, err := device.CreateDepthStencilView(texture.Resource())
	if err != nil {
		texture.Release()
		return nil, nil, err
	}
	return texture, view, nil
}

func setRasterizerState(device *d3d.ID3D11Device, deviceContext *d3d.ID3D11DeviceContext) error {
	desc := d3d.D3D11_RASTERIZER_DESC{
		FillMode:              d3d.D3D11_FILL_SOLID,
		CullMode:              d3d.D3D11_CULL_NONE,
		FrontCounterClockwise: d3d.BoolOf(false),
		DepthBias:             0,
		DepthBiasClamp:        0.0,
		SlopeScaledDepthBias:  0.0,
		DepthClipEnable:       d3d.BoolOf(true),
		ScissorEnable:         d3d.BoolOf(false),
		MultisampleEnable:     d3d.BoolOf(true),
		AntialiasedLineEnable: d3d.BoolOf(false),
	}
	rasterizerState, err := device.CreateRasterizerState(&desc)
	if err != nil {
		return err
	}
	deviceContext.RSSetState(rasterizerState)
	rasterizerState.Release()
	return nil
}

func createBlendState(device *d3d.ID3D11Device) (*d3d.ID3D11BlendState, error) {
	var desc d3d.D3D11_BLEND_DESC
	desc.RenderTarget[0].BlendEnable = d3d.BoolOf(true)
	desc.RenderTarget[0].BlendOp = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].BlendOpAlpha = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].SrcBlend = d3d.D3D11_BLEND_SRC_ALPHA
	desc.RenderTarget[0].SrcBlendAlpha = d3d.D3D11_BLEND_ONE
	desc.RenderTarget[0].DestBlend = d3d.D3D11_BLEND_INV_SRC_ALPHA
	desc.RenderTarget[0].DestBlendAlpha = d3d.D3D11_BLEND_INV_SRC_ALPHA
	desc.RenderTarget[0].RenderTargetWriteMask = d3d.D3D11_COLOR_WRITE_ENABLE_ALL
	return device.CreateBlendState(&desc)
}

func createOpaqueBlendState(device *d3d.ID3D11Device) (*d3d.ID3D11BlendState, error) {
	var desc d3d.D3D11_BLEND_DESC
	desc.RenderTarget[0].BlendEnable = d3d.BoolOf(false)
	desc.RenderTarget[0].BlendOp = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].BlendOpAlpha = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].SrcBlend = d3d.D3D11_BLEND_ONE
	desc.RenderTarget[0].SrcBlendAlpha = d3d.D3D11_BLEND_ONE
	desc.RenderTarget[0].DestBlend = d3d.D3D11_BLEND_ZERO
	desc.RenderTarget[0].DestBlendAlpha = d3d.D3D11_BLEND_ZERO
	desc.RenderTarget[0].RenderTargetWriteMask = d3d.D3D11_COLOR_WRITE_ENABLE_ALL
	return device.CreateBlendState(&desc)
}

func createDepthStencilState(device *d3d.ID3D11Device, writeMask int32) (*d3d.ID3D11DepthStencilState, error) {
	stencilOp := d3d.D3D11_DEPTH_STENCILOP_DESC{
		StencilFailOp:      d3d.D3D11_STENCIL_OP_KEEP,
		StencilDepthFailOp: d3d.D3D11_STENCIL_OP_KEEP,
		StencilPassOp:      d3d.D3D11_STENCIL_OP_KEEP,
		StencilFunc:        d3d.D3D11_COMPARISON_ALWAYS,
	}
	desc := d3d.D3D11_DEPTH_STENCIL_DESC{
		DepthEnable:      d3d.BoolOf(true),
		DepthWriteMask:   writeMask,
		DepthFunc:        d3d.D3D11_COMPARISON_GREATER,
		StencilEnable:    d3d.BoolOf(false),
		StencilReadMask:  d3d.D3D11_DEFAULT_STENCIL_READ_MASK,
		StencilWriteMask: d3d.D3D11_DEFAULT_STENCIL_WRITE_MASK,
		FrontFace:        stencilOp,
		BackFace:         stencilOp,
	}
	return device.CreateDepthStencilState(&desc)
}

func createBlendStateForSubpixelRendering(device *d3d.ID3D11Device) (*d3d.ID3D11BlendState, error) {
	var desc d3d.D3D11_BLEND_DESC
	desc.RenderTarget[0].BlendEnable = d3d.BoolOf(true)
	desc.RenderTarget[0].BlendOp = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].BlendOpAlpha = d3d.D3D11_BLEND_OP_ADD
	desc.RenderTarget[0].SrcBlend = d3d.D3D11_BLEND_SRC1_COLOR
	desc.RenderTarget[0].DestBlend = d3d.D3D11_BLEND_INV_SRC1_COLOR
	desc.RenderTarget[0].SrcBlendAlpha = d3d.D3D11_BLEND_ONE
	desc.RenderTarget[0].DestBlendAlpha = d3d.D3D11_BLEND_ZERO
	desc.RenderTarget[0].RenderTargetWriteMask = d3d.D3D11_COLOR_WRITE_ENABLE_ALL &^ d3d.D3D11_COLOR_WRITE_ENABLE_ALPHA
	return device.CreateBlendState(&desc)
}

func createVertexShader(device *d3d.ID3D11Device, bytes []byte) (*d3d.ID3D11VertexShader, error) {
	return device.CreateVertexShader(bytes)
}

func createFragmentShader(device *d3d.ID3D11Device, bytes []byte) (*d3d.ID3D11PixelShader, error) {
	return device.CreatePixelShader(bytes)
}

func createConstantBuffer[T any](device *d3d.ID3D11Device) (*d3d.ID3D11Buffer, error) {
	size := sizeOf[T]()
	if size == 0 || size%16 != 0 {
		panic("assertion failed: std::mem::size_of::<T>() != 0 && std::mem::size_of::<T>().is_multiple_of(16)")
	}
	desc := d3d.D3D11_BUFFER_DESC{
		ByteWidth:           uint32(size),
		Usage:               d3d.D3D11_USAGE_DYNAMIC,
		BindFlags:           d3d.D3D11_BIND_CONSTANT_BUFFER,
		CPUAccessFlags:      d3d.D3D11_CPU_ACCESS_WRITE,
		MiscFlags:           0,
		StructureByteStride: 0,
	}
	return device.CreateBuffer(&desc, nil)
}

func createBuffer(device *d3d.ID3D11Device, elementSize, bufferSize int) (*d3d.ID3D11Buffer, error) {
	hi, lo := bits.Mul64(uint64(elementSize), uint64(bufferSize))
	if hi != 0 || lo > 0xFFFFFFFF {
		return nil, errors.New("structured buffer size overflows u32")
	}
	if uint64(elementSize) > 0xFFFFFFFF {
		return nil, errors.New("structured buffer stride overflows u32")
	}
	desc := d3d.D3D11_BUFFER_DESC{
		ByteWidth:           uint32(lo),
		Usage:               d3d.D3D11_USAGE_DYNAMIC,
		BindFlags:           d3d.D3D11_BIND_SHADER_RESOURCE,
		CPUAccessFlags:      d3d.D3D11_CPU_ACCESS_WRITE,
		MiscFlags:           d3d.D3D11_RESOURCE_MISC_BUFFER_STRUCTURED,
		StructureByteStride: uint32(elementSize),
	}
	buffer, err := device.CreateBuffer(&desc, nil)
	if err != nil {
		return nil, err
	}
	if buffer == nil {
		panic("called `Option::unwrap()` on a `None` value")
	}
	return buffer, nil
}

func createBufferView(device *d3d.ID3D11Device, buffer *d3d.ID3D11Buffer) (*d3d.ID3D11ShaderResourceView, error) {
	return device.CreateShaderResourceView(buffer.Resource())
}

func updateBuffer[T any](deviceContext *d3d.ID3D11DeviceContext, buffer *d3d.ID3D11Buffer, data []T) error {
	dest, err := deviceContext.Map(buffer.Resource(), 0, d3d.D3D11_MAP_WRITE_DISCARD, 0)
	if err != nil {
		return err
	}
	if len(data) > 0 {
		n := len(data) * sizeOf[T]()
		src := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(data))), n)
		copy(unsafe.Slice((*byte)(dest.PData), n), src)
	}
	deviceContext.Unmap(buffer.Resource(), 0)
	return nil
}

func updateBatchStart(deviceContext *d3d.ID3D11DeviceContext, buffer *d3d.ID3D11Buffer, firstInstance uint32) error {
	return updateBuffer(deviceContext, buffer, []batchParams{{startIndex: firstInstance}})
}

func setPipelineState(deviceContext *d3d.ID3D11DeviceContext, bufferView []*d3d.ID3D11ShaderResourceView, topology uint32, vertexShader *d3d.ID3D11VertexShader, fragmentShader *d3d.ID3D11PixelShader, blendState *d3d.ID3D11BlendState, depthStencilState *d3d.ID3D11DepthStencilState) {
	deviceContext.VSSetShaderResources(1, bufferView)
	deviceContext.PSSetShaderResources(1, bufferView)
	deviceContext.IASetPrimitiveTopology(topology)
	deviceContext.VSSetShader(vertexShader)
	deviceContext.PSSetShader(fragmentShader)
	deviceContext.OMSetBlendState(blendState, nil, 0xFFFFFFFF)
	deviceContext.OMSetDepthStencilState(depthStencilState, 0)
}

func reportLiveObjects(device *d3d.ID3D11Device) error {
	debugDevice, err := device.CastDebug()
	if err != nil {
		return err
	}
	defer debugDevice.Release()
	return debugDevice.ReportLiveDeviceObjects(d3d.D3D11_RLDO_DETAIL)
}

const nvapiShortStringMax = 64

func nvidiaGetDriverVersion() (string, error) {
	nvidiaDllName := "nvapi64.dll"
	if unsafe.Sizeof(uintptr(0)) == 4 {
		nvidiaDllName = "nvapi.dll"
	}
	return withDllLibrary(nvidiaDllName, func(nvidiaDll win32.HMODULE) (string, error) {
		nvapiQueryAddr, err := xwin.GetProcAddress(nvidiaDll, "nvapi_QueryInterface")
		if err != nil || nvapiQueryAddr == 0 {
			return "", errors.New("Failed to get nvapi_QueryInterface address")
		}
		nvapiGetDriverVersionPtr, _, _ := syscall.SyscallN(nvapiQueryAddr, 0x2926aaad)
		if nvapiGetDriverVersionPtr == 0 {
			return "", errors.New("Failed to get NVIDIA driver version function pointer")
		}
		var driverVersion uint32
		var buildBranchString [nvapiShortStringMax]byte
		result, _, _ := syscall.SyscallN(nvapiGetDriverVersionPtr,
			uintptr(unsafe.Pointer(&driverVersion)),
			uintptr(unsafe.Pointer(&buildBranchString)),
		)
		if int32(result) != 0 {
			return "", fmt.Errorf("Failed to get NVIDIA driver version, error code: %d", int32(result))
		}
		major := driverVersion / 100
		minor := driverVersion % 100
		branchString := xwin.ByteSliceToString(buildBranchString[:])
		return fmt.Sprintf("%d.%d %s", major, minor, branchString), nil
	})
}

const agsCurrentVersion int32 = (6 << 22) | (3 << 12)

// AGSGPUInfo is the GPU information AMD AGS fills in agsInitialize.
type AGSGPUInfo struct {
	DriverVersion         uintptr
	RadeonSoftwareVersion uintptr
	NumDevices            int32
	Devices               uintptr
}

func amdGetDriverVersion() (string, error) {
	amdDllName := "amd_ags_x64.dll"
	if unsafe.Sizeof(uintptr(0)) == 4 {
		amdDllName = "amd_ags_x86.dll"
	}
	return withDllLibrary(amdDllName, func(amdDll win32.HMODULE) (string, error) {
		agsInitializeAddr, err := xwin.GetProcAddress(amdDll, "agsInitialize")
		if err != nil || agsInitializeAddr == 0 {
			return "", errors.New("Failed to get agsInitialize address")
		}
		agsDeinitializeAddr, err := xwin.GetProcAddress(amdDll, "agsDeInitialize")
		if err != nil || agsDeinitializeAddr == 0 {
			return "", errors.New("Failed to get agsDeInitialize address")
		}
		var context uintptr
		var gpuInfo AGSGPUInfo
		result, _, _ := syscall.SyscallN(agsInitializeAddr,
			uintptr(agsCurrentVersion),
			0,
			uintptr(unsafe.Pointer(&context)),
			uintptr(unsafe.Pointer(&gpuInfo)),
		)
		if int32(result) != 0 {
			return "", fmt.Errorf("Failed to initialize AMD AGS, error code: %d", int32(result))
		}
		softwareVersion := "Unknown Radeon Software Version"
		if gpuInfo.RadeonSoftwareVersion != 0 {
			softwareVersion = xwin.BytePtrToString((*byte)(com.PtrFromUintptr(gpuInfo.RadeonSoftwareVersion)))
		}
		driverVersion := "Unknown Radeon Driver Version"
		if gpuInfo.DriverVersion != 0 {
			driverVersion = xwin.BytePtrToString((*byte)(com.PtrFromUintptr(gpuInfo.DriverVersion)))
		}
		_, _, _ = syscall.SyscallN(agsDeinitializeAddr, context)
		return fmt.Sprintf("%s (%s)", softwareVersion, driverVersion), nil
	})
}

func dxgiGetDriverVersion(adapter *d3d.IDXGIAdapter1) (string, error) {
	number, err := adapter.CheckInterfaceSupport(&d3d.IID_IDXGIDevice)
	if err != nil {
		return "", err
	}
	n := uint64(number)
	return fmt.Sprintf("%d.%d.%d.%d", n>>48, (n>>32)&0xFFFF, (n>>16)&0xFFFF, n&0xFFFF), nil
}
