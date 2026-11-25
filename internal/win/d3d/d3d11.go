//go:build windows

package d3d

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	modD3D11              = windows.NewLazySystemDLL("d3d11.dll")
	procD3D11CreateDevice = modD3D11.NewProc("D3D11CreateDevice")
)

func iptr[T any](p *T) uintptr {
	return uintptr(unsafe.Pointer(p))
}

// D3D11CreateDevice creates a device and its immediate context. adapter may be nil; the returned
// feature level is the one the device was created with.
func D3D11CreateDevice(adapter *IDXGIAdapter1, driverType uint32, software windows.Handle, flags uint32, featureLevels []uint32, sdkVersion uint32) (*ID3D11Device, uint32, *ID3D11DeviceContext, error) {
	if err := procD3D11CreateDevice.Find(); err != nil {
		return nil, 0, nil, err
	}
	var device *ID3D11Device
	var context *ID3D11DeviceContext
	var featureLevel uint32
	r, _, _ := procD3D11CreateDevice.Call(
		iptr(adapter),
		uintptr(driverType),
		uintptr(software),
		uintptr(flags),
		uintptr(slicePtr(featureLevels)),
		uintptr(len(featureLevels)),
		uintptr(sdkVersion),
		uintptr(unsafe.Pointer(&device)),
		uintptr(unsafe.Pointer(&featureLevel)),
		uintptr(unsafe.Pointer(&context)),
	)
	runtime.KeepAlive(featureLevels)
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, 0, nil, err
	}
	return device, featureLevel, context, nil
}

const (
	deviceChildSlots = com.IUnknownSlots + 4
	resourceSlots    = deviceChildSlots + 3
	viewSlots        = deviceChildSlots + 1
)

// ID3D11Device is a Direct3D 11 device.
type ID3D11Device struct{ com.IUnknown }

// Clone adds a reference and returns d.
func (d *ID3D11Device) Clone() *ID3D11Device {
	d.AddRef()
	return d
}

// CreateBuffer creates a buffer; initialData may be nil.
func (d *ID3D11Device) CreateBuffer(desc *D3D11_BUFFER_DESC, initialData *D3D11_SUBRESOURCE_DATA) (*ID3D11Buffer, error) {
	var out *ID3D11Buffer
	err := com.CallHR(d.Ptr(), 3, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(initialData)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateTexture2D creates a 2D texture; initialData may be nil.
func (d *ID3D11Device) CreateTexture2D(desc *D3D11_TEXTURE2D_DESC, initialData *D3D11_SUBRESOURCE_DATA) (*ID3D11Texture2D, error) {
	var out *ID3D11Texture2D
	err := com.CallHR(d.Ptr(), 5, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(initialData)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateShaderResourceView creates a shader resource view of resource with the default description.
func (d *ID3D11Device) CreateShaderResourceView(resource *ID3D11Resource) (*ID3D11ShaderResourceView, error) {
	var out *ID3D11ShaderResourceView
	err := com.CallHR(d.Ptr(), 7, iptr(resource), 0, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateRenderTargetView creates a render target view of resource with the default description.
func (d *ID3D11Device) CreateRenderTargetView(resource *ID3D11Resource) (*ID3D11RenderTargetView, error) {
	var out *ID3D11RenderTargetView
	err := com.CallHR(d.Ptr(), 9, iptr(resource), 0, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateDepthStencilView creates a depth-stencil view of resource with the default description.
func (d *ID3D11Device) CreateDepthStencilView(resource *ID3D11Resource) (*ID3D11DepthStencilView, error) {
	var out *ID3D11DepthStencilView
	err := com.CallHR(d.Ptr(), 10, iptr(resource), 0, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateVertexShader creates a vertex shader from compiled bytecode.
func (d *ID3D11Device) CreateVertexShader(bytecode []byte) (*ID3D11VertexShader, error) {
	var out *ID3D11VertexShader
	err := com.CallHR(d.Ptr(), 12, uintptr(slicePtr(bytecode)), uintptr(len(bytecode)), 0, uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(bytecode)
	return out, err
}

// CreatePixelShader creates a pixel shader from compiled bytecode.
func (d *ID3D11Device) CreatePixelShader(bytecode []byte) (*ID3D11PixelShader, error) {
	var out *ID3D11PixelShader
	err := com.CallHR(d.Ptr(), 15, uintptr(slicePtr(bytecode)), uintptr(len(bytecode)), 0, uintptr(unsafe.Pointer(&out)))
	runtime.KeepAlive(bytecode)
	return out, err
}

// CreateBlendState creates a blend state.
func (d *ID3D11Device) CreateBlendState(desc *D3D11_BLEND_DESC) (*ID3D11BlendState, error) {
	var out *ID3D11BlendState
	err := com.CallHR(d.Ptr(), 20, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateDepthStencilState creates a depth-stencil state.
func (d *ID3D11Device) CreateDepthStencilState(desc *D3D11_DEPTH_STENCIL_DESC) (*ID3D11DepthStencilState, error) {
	var out *ID3D11DepthStencilState
	err := com.CallHR(d.Ptr(), 21, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateRasterizerState creates a rasterizer state.
func (d *ID3D11Device) CreateRasterizerState(desc *D3D11_RASTERIZER_DESC) (*ID3D11RasterizerState, error) {
	var out *ID3D11RasterizerState
	err := com.CallHR(d.Ptr(), 22, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateSamplerState creates a sampler state.
func (d *ID3D11Device) CreateSamplerState(desc *D3D11_SAMPLER_DESC) (*ID3D11SamplerState, error) {
	var out *ID3D11SamplerState
	err := com.CallHR(d.Ptr(), 23, uintptr(unsafe.Pointer(desc)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CheckFeatureSupport fills data, of size bytes, with the device's support for feature.
func (d *ID3D11Device) CheckFeatureSupport(feature uint32, data unsafe.Pointer, size uint32) error {
	return com.CallHR(d.Ptr(), 33, uintptr(feature), uintptr(data), uintptr(size))
}

// GetFeatureLevel returns the device's feature level.
func (d *ID3D11Device) GetFeatureLevel() uint32 {
	return uint32(com.Call(d.Ptr(), 37))
}

// GetDeviceRemovedReason returns why the device was removed, or nil if it was not.
func (d *ID3D11Device) GetDeviceRemovedReason() error {
	return com.HRESULTFromUintptr(com.Call(d.Ptr(), 39)).Err()
}

// GetImmediateContext returns the device's immediate context, AddRef'd.
func (d *ID3D11Device) GetImmediateContext() *ID3D11DeviceContext {
	var out *ID3D11DeviceContext
	com.Call(d.Ptr(), 40, uintptr(unsafe.Pointer(&out)))
	return out
}

// CastDXGIDevice queries the device for IDXGIDevice.
func (d *ID3D11Device) CastDXGIDevice() (*IDXGIDevice, error) {
	return com.Cast[IDXGIDevice](&d.IUnknown, &IID_IDXGIDevice)
}

// CastDebug queries the device for ID3D11Debug.
func (d *ID3D11Device) CastDebug() (*ID3D11Debug, error) {
	return com.Cast[ID3D11Debug](&d.IUnknown, &IID_ID3D11Debug)
}

// ID3D11DeviceContext is a device context.
type ID3D11DeviceContext struct{ com.IUnknown }

// Clone adds a reference and returns c.
func (c *ID3D11DeviceContext) Clone() *ID3D11DeviceContext {
	c.AddRef()
	return c
}

// VSSetConstantBuffers binds constant buffers to the vertex shader stage.
func (c *ID3D11DeviceContext) VSSetConstantBuffers(startSlot uint32, buffers []*ID3D11Buffer) {
	com.Call(c.Ptr(), 7, uintptr(startSlot), uintptr(len(buffers)), uintptr(slicePtr(buffers)))
	runtime.KeepAlive(buffers)
}

// PSSetShaderResources binds shader resource views to the pixel shader stage.
func (c *ID3D11DeviceContext) PSSetShaderResources(startSlot uint32, views []*ID3D11ShaderResourceView) {
	com.Call(c.Ptr(), 8, uintptr(startSlot), uintptr(len(views)), uintptr(slicePtr(views)))
	runtime.KeepAlive(views)
}

// PSSetShader sets the pixel shader.
func (c *ID3D11DeviceContext) PSSetShader(shader *ID3D11PixelShader) {
	com.Call(c.Ptr(), 9, iptr(shader), 0, 0)
}

// PSSetSamplers binds samplers to the pixel shader stage.
func (c *ID3D11DeviceContext) PSSetSamplers(startSlot uint32, samplers []*ID3D11SamplerState) {
	com.Call(c.Ptr(), 10, uintptr(startSlot), uintptr(len(samplers)), uintptr(slicePtr(samplers)))
	runtime.KeepAlive(samplers)
}

// VSSetShader sets the vertex shader.
func (c *ID3D11DeviceContext) VSSetShader(shader *ID3D11VertexShader) {
	com.Call(c.Ptr(), 11, iptr(shader), 0, 0)
}

// Draw draws non-indexed primitives.
func (c *ID3D11DeviceContext) Draw(vertexCount, startVertexLocation uint32) {
	com.Call(c.Ptr(), 13, uintptr(vertexCount), uintptr(startVertexLocation))
}

// Map maps a subresource for CPU access.
func (c *ID3D11DeviceContext) Map(resource *ID3D11Resource, subresource, mapType, mapFlags uint32) (D3D11_MAPPED_SUBRESOURCE, error) {
	var mapped D3D11_MAPPED_SUBRESOURCE
	err := com.CallHR(c.Ptr(), 14, iptr(resource), uintptr(subresource), uintptr(mapType), uintptr(mapFlags), uintptr(unsafe.Pointer(&mapped)))
	return mapped, err
}

// Unmap unmaps a subresource.
func (c *ID3D11DeviceContext) Unmap(resource *ID3D11Resource, subresource uint32) {
	com.Call(c.Ptr(), 15, iptr(resource), uintptr(subresource))
}

// PSSetConstantBuffers binds constant buffers to the pixel shader stage.
func (c *ID3D11DeviceContext) PSSetConstantBuffers(startSlot uint32, buffers []*ID3D11Buffer) {
	com.Call(c.Ptr(), 16, uintptr(startSlot), uintptr(len(buffers)), uintptr(slicePtr(buffers)))
	runtime.KeepAlive(buffers)
}

// DrawInstanced draws instanced, non-indexed primitives.
func (c *ID3D11DeviceContext) DrawInstanced(vertexCountPerInstance, instanceCount, startVertexLocation, startInstanceLocation uint32) {
	com.Call(c.Ptr(), 21, uintptr(vertexCountPerInstance), uintptr(instanceCount), uintptr(startVertexLocation), uintptr(startInstanceLocation))
}

// IASetPrimitiveTopology sets the primitive topology.
func (c *ID3D11DeviceContext) IASetPrimitiveTopology(topology uint32) {
	com.Call(c.Ptr(), 24, uintptr(topology))
}

// VSSetShaderResources binds shader resource views to the vertex shader stage.
func (c *ID3D11DeviceContext) VSSetShaderResources(startSlot uint32, views []*ID3D11ShaderResourceView) {
	com.Call(c.Ptr(), 25, uintptr(startSlot), uintptr(len(views)), uintptr(slicePtr(views)))
	runtime.KeepAlive(views)
}

// OMSetRenderTargets binds render targets and a depth-stencil view (which may be nil).
func (c *ID3D11DeviceContext) OMSetRenderTargets(views []*ID3D11RenderTargetView, depthStencilView *ID3D11DepthStencilView) {
	com.Call(c.Ptr(), 33, uintptr(len(views)), uintptr(slicePtr(views)), iptr(depthStencilView))
	runtime.KeepAlive(views)
}

// OMSetBlendState sets the blend state; blendFactor may be nil.
func (c *ID3D11DeviceContext) OMSetBlendState(state *ID3D11BlendState, blendFactor *[4]float32, sampleMask uint32) {
	com.Call(c.Ptr(), 35, iptr(state), uintptr(unsafe.Pointer(blendFactor)), uintptr(sampleMask))
}

// OMSetDepthStencilState sets the depth-stencil state.
func (c *ID3D11DeviceContext) OMSetDepthStencilState(state *ID3D11DepthStencilState, stencilRef uint32) {
	com.Call(c.Ptr(), 36, iptr(state), uintptr(stencilRef))
}

// RSSetState sets the rasterizer state.
func (c *ID3D11DeviceContext) RSSetState(state *ID3D11RasterizerState) {
	com.Call(c.Ptr(), 43, iptr(state))
}

// RSSetViewports sets the viewports.
func (c *ID3D11DeviceContext) RSSetViewports(viewports []D3D11_VIEWPORT) {
	com.Call(c.Ptr(), 44, uintptr(len(viewports)), uintptr(slicePtr(viewports)))
	runtime.KeepAlive(viewports)
}

// RSSetScissorRects sets the scissor rectangles.
func (c *ID3D11DeviceContext) RSSetScissorRects(rects []RECT) {
	com.Call(c.Ptr(), 45, uintptr(len(rects)), uintptr(slicePtr(rects)))
	runtime.KeepAlive(rects)
}

// CopySubresourceRegion copies a region of src into dst; srcBox may be nil.
func (c *ID3D11DeviceContext) CopySubresourceRegion(dst *ID3D11Resource, dstSubresource, dstX, dstY, dstZ uint32, src *ID3D11Resource, srcSubresource uint32, srcBox *D3D11_BOX) {
	com.Call(c.Ptr(), 46, iptr(dst), uintptr(dstSubresource), uintptr(dstX), uintptr(dstY), uintptr(dstZ), iptr(src), uintptr(srcSubresource), uintptr(unsafe.Pointer(srcBox)))
}

// CopyResource copies src into dst.
func (c *ID3D11DeviceContext) CopyResource(dst, src *ID3D11Resource) {
	com.Call(c.Ptr(), 47, iptr(dst), iptr(src))
}

// UpdateSubresource copies data from CPU memory into a subresource; box may be nil.
func (c *ID3D11DeviceContext) UpdateSubresource(dst *ID3D11Resource, dstSubresource uint32, box *D3D11_BOX, data unsafe.Pointer, rowPitch, depthPitch uint32) {
	com.Call(c.Ptr(), 48, iptr(dst), uintptr(dstSubresource), uintptr(unsafe.Pointer(box)), uintptr(data), uintptr(rowPitch), uintptr(depthPitch))
}

// ClearRenderTargetView fills a render target with color.
func (c *ID3D11DeviceContext) ClearRenderTargetView(view *ID3D11RenderTargetView, color *[4]float32) {
	com.Call(c.Ptr(), 50, iptr(view), uintptr(unsafe.Pointer(color)))
}

// ClearDepthStencilView clears a depth-stencil view.
func (c *ID3D11DeviceContext) ClearDepthStencilView(view *ID3D11DepthStencilView, clearFlags uint32, depth float32, stencil uint8) {
	call := floatFn[func(unsafe.Pointer, unsafe.Pointer, uint32, float32, uint8)](c.Ptr(), 53)
	call(c.Ptr(), unsafe.Pointer(view), clearFlags, depth, stencil)
}

// ClearState resets the context to its default state.
func (c *ID3D11DeviceContext) ClearState() {
	com.Call(c.Ptr(), 110)
}

// Flush sends queued commands to the GPU.
func (c *ID3D11DeviceContext) Flush() {
	com.Call(c.Ptr(), 111)
}

// CastDeviceContext1 queries the context for ID3D11DeviceContext1.
func (c *ID3D11DeviceContext) CastDeviceContext1() (*ID3D11DeviceContext1, error) {
	return com.Cast[ID3D11DeviceContext1](&c.IUnknown, &IID_ID3D11DeviceContext1)
}

// CastUserDefinedAnnotation queries the context for ID3DUserDefinedAnnotation.
func (c *ID3D11DeviceContext) CastUserDefinedAnnotation() (*ID3DUserDefinedAnnotation, error) {
	return com.Cast[ID3DUserDefinedAnnotation](&c.IUnknown, &IID_ID3DUserDefinedAnnotation)
}

// ID3D11DeviceContext1 is a Direct3D 11.1 device context.
type ID3D11DeviceContext1 struct{ ID3D11DeviceContext }

// DiscardResource discards a resource's contents.
func (c *ID3D11DeviceContext1) DiscardResource(resource *ID3D11Resource) {
	com.Call(c.Ptr(), 117, iptr(resource))
}

// DiscardView discards the contents of a view (an ID3D11View such as a depth-stencil view).
func (c *ID3D11DeviceContext1) DiscardView(view *com.IUnknown) {
	com.Call(c.Ptr(), 118, iptr(view))
}

// ID3D11Resource is a resource.
type ID3D11Resource struct{ com.IUnknown }

// ID3D11Texture2D is a 2D texture.
type ID3D11Texture2D struct{ ID3D11Resource }

// Clone adds a reference and returns t.
func (t *ID3D11Texture2D) Clone() *ID3D11Texture2D {
	t.AddRef()
	return t
}

// Resource returns t as an ID3D11Resource.
func (t *ID3D11Texture2D) Resource() *ID3D11Resource {
	return &t.ID3D11Resource
}

// GetDesc returns the texture's description.
func (t *ID3D11Texture2D) GetDesc() D3D11_TEXTURE2D_DESC {
	var desc D3D11_TEXTURE2D_DESC
	com.Call(t.Ptr(), resourceSlots, uintptr(unsafe.Pointer(&desc)))
	return desc
}

// ID3D11Buffer is a buffer.
type ID3D11Buffer struct{ ID3D11Resource }

// Resource returns b as an ID3D11Resource.
func (b *ID3D11Buffer) Resource() *ID3D11Resource {
	return &b.ID3D11Resource
}

// GetDesc returns the buffer's description.
func (b *ID3D11Buffer) GetDesc() D3D11_BUFFER_DESC {
	var desc D3D11_BUFFER_DESC
	com.Call(b.Ptr(), resourceSlots, uintptr(unsafe.Pointer(&desc)))
	return desc
}

// ID3D11View is a resource view.
type ID3D11View struct{ com.IUnknown }

// GetResource returns the viewed resource, AddRef'd.
func (v *ID3D11View) GetResource() *ID3D11Resource {
	var out *ID3D11Resource
	com.Call(v.Ptr(), deviceChildSlots, uintptr(unsafe.Pointer(&out)))
	return out
}

// ID3D11ShaderResourceView is a shader resource view.
type ID3D11ShaderResourceView struct{ ID3D11View }

// Clone adds a reference and returns v.
func (v *ID3D11ShaderResourceView) Clone() *ID3D11ShaderResourceView {
	v.AddRef()
	return v
}

// ID3D11RenderTargetView is a render target view.
type ID3D11RenderTargetView struct{ ID3D11View }

// ID3D11DepthStencilView is a depth-stencil view.
type ID3D11DepthStencilView struct{ ID3D11View }

// ID3D11VertexShader is a vertex shader.
type ID3D11VertexShader struct{ com.IUnknown }

// ID3D11PixelShader is a pixel shader.
type ID3D11PixelShader struct{ com.IUnknown }

// ID3D11BlendState is a blend state.
type ID3D11BlendState struct{ com.IUnknown }

// ID3D11DepthStencilState is a depth-stencil state.
type ID3D11DepthStencilState struct{ com.IUnknown }

// ID3D11RasterizerState is a rasterizer state.
type ID3D11RasterizerState struct{ com.IUnknown }

// ID3D11SamplerState is a sampler state.
type ID3D11SamplerState struct{ com.IUnknown }

// ID3D11Debug is the debug layer interface of a device.
type ID3D11Debug struct{ com.IUnknown }

// ReportLiveDeviceObjects reports the device's live objects to the debug output.
func (d *ID3D11Debug) ReportLiveDeviceObjects(flags int32) error {
	return com.CallHR(d.Ptr(), com.IUnknownSlots+7, uintptr(flags))
}

// ID3DUserDefinedAnnotation marks GPU events for graphics debuggers.
type ID3DUserDefinedAnnotation struct{ com.IUnknown }

// BeginEvent opens a named event.
func (a *ID3DUserDefinedAnnotation) BeginEvent(name *uint16) int32 {
	return int32(com.Call(a.Ptr(), com.IUnknownSlots, uintptr(unsafe.Pointer(name))))
}

// EndEvent closes the innermost event.
func (a *ID3DUserDefinedAnnotation) EndEvent() int32 {
	return int32(com.Call(a.Ptr(), com.IUnknownSlots+1))
}

// GetStatus reports whether a graphics debugger is capturing.
func (a *ID3DUserDefinedAnnotation) GetStatus() bool {
	return BOOL(int32(com.Call(a.Ptr(), com.IUnknownSlots+3))).AsBool()
}
