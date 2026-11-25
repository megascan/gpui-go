//go:build js && wasm

package webgl2

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"syscall/js"

	"github.com/megascan/gpui-go/internal/webglsl"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

const (
	maxTextureSlots = 16
	maxSamplers     = 16
)

// CreateInstance creates a WebGL2 instance. Adapters come from the WebGL2 context of the
// last canvas surface created on it, as wgpu-hal's web Instance.
func CreateInstance(gpu.InstanceDescriptor) gpu.Instance {
	return &instance{}
}

type instance struct {
	context glContext
}

func (i *instance) EnumerateAdapters(backends gpu.Backends) []gpu.Adapter {
	if backends&gpu.BackendsGl == 0 || i.context == nil {
		return nil
	}
	return []gpu.Adapter{exposeAdapter(i.context)}
}

func (i *instance) RequestAdapter(options *gpu.RequestAdapterOptions) (gpu.Adapter, error) {
	gl := i.context
	if options != nil && options.CompatibleSurface != nil {
		s, ok := options.CompatibleSurface.(*surface)
		if !ok {
			return nil, errors.New("compatible surface is not a WebGL2 surface")
		}
		gl = s.gl
	}
	if gl == nil {
		return nil, errors.New("no suitable adapter found: create a canvas surface first")
	}
	return exposeAdapter(gl), nil
}

func (i *instance) CreateSurface(target gpu.SurfaceTarget) (gpu.Surface, error) {
	if target.CanvasID == "" {
		return nil, errors.New("unsupported surface target")
	}
	canvas := js.Global().Get("document").Call("getElementById", target.CanvasID)
	if !canvas.Truthy() {
		return nil, fmt.Errorf("no canvas element with id %q", target.CanvasID)
	}
	return i.createSurfaceFromCanvas(canvas)
}

func (i *instance) Release() {}

type adapter struct {
	gl     glContext
	info   gpu.AdapterInfo
	limits gpu.Limits
}

var integratedRendererHints = []string{
	" xpress", "amd renoir", "radeon hd 4200", "radeon hd 4250", "radeon hd 4290", "radeon hd 4270",
	"radeon hd 4225", "radeon hd 3100", "radeon hd 3200", "radeon hd 3000", "radeon hd 3300",
	"radeon(tm) r4 graphics", "radeon(tm) r5 graphics", "radeon(tm) r6 graphics", "radeon(tm) r7 graphics",
	"radeon r7 graphics", "nforce", "tegra", "shield", "igp", "mali", "intel", "v3d", "apple m",
}

var cpuRendererHints = []string{"mesa offscreen", "swiftshader", "llvmpipe"}

var vendorIDs = []struct {
	name string
	id   uint32
}{
	{"amd", 0x1002}, {"imgtec", 0x1010}, {"nvidia", 0x10DE}, {"arm", 0x13B5}, {"qualcomm", 0x5143},
	{"intel", 0x8086}, {"broadcom", 0x14E4}, {"mesa", 0x10005}, {"apple", 0x106B},
}

func makeInfo(vendorOrig, rendererOrig, version string) gpu.AdapterInfo {
	vendor := strings.ToLower(vendorOrig)
	renderer := strings.ToLower(rendererOrig)
	deviceType := gpu.DeviceTypeOther
	integrated := strings.Contains(vendor, "qualcomm") || strings.Contains(vendor, "intel")
	for _, s := range integratedRendererHints {
		integrated = integrated || strings.Contains(renderer, s)
	}
	if integrated {
		deviceType = gpu.DeviceTypeIntegratedGpu
	} else {
		for _, s := range cpuRendererHints {
			if strings.Contains(renderer, s) {
				deviceType = gpu.DeviceTypeCpu
				break
			}
		}
	}
	var vendorID uint32
	for _, v := range vendorIDs {
		if strings.Contains(vendor, v.name) {
			vendorID = v.id
			break
		}
	}
	return gpu.AdapterInfo{
		Name:       rendererOrig,
		Vendor:     vendorID,
		DeviceType: deviceType,
		DriverInfo: version,
		Backend:    gpu.BackendGl,
	}
}

func exposeAdapter(gl glContext) *adapter {
	vendorConst, rendererConst := uint32(glVendor), uint32(glRenderer)
	for _, ext := range gl.supportedExtensions() {
		if ext == "WEBGL_debug_renderer_info" {
			if gl.enableExtension(ext) {
				vendorConst, rendererConst = glUnmaskedVendorWebGL, glUnmaskedRendererWebGL
			}
			break
		}
	}
	info := makeInfo(gl.getParameterString(vendorConst), gl.getParameterString(rendererConst), gl.getParameterString(glVersion))
	return &adapter{gl: gl, info: info, limits: queryLimits(gl)}
}

func queryLimits(gl glContext) gpu.Limits {
	l := gpu.DownlevelWebGL2Defaults()
	positive := func(v int, fallback uint32) uint32 {
		if v <= 0 {
			return fallback
		}
		return uint32(v)
	}
	maxTextureSize := positive(gl.getParameterInt(glMaxTextureSize), l.MaxTextureDimension2D)
	l.MaxTextureDimension1D = maxTextureSize
	l.MaxTextureDimension2D = maxTextureSize
	l.MaxTextureDimension3D = positive(gl.getParameterInt(glMax3DTextureSize), l.MaxTextureDimension3D)
	l.MaxTextureArrayLayers = positive(gl.getParameterInt(glMaxArrayTextureLayers), l.MaxTextureArrayLayers)
	l.MinUniformBufferOffsetAlignment = positive(gl.getParameterInt(glUniformBufferOffsetAlign), l.MinUniformBufferOffsetAlignment)
	l.MinStorageBufferOffsetAlignment = 256
	l.MaxUniformBuffersPerShaderStage = min(
		positive(gl.getParameterInt(glMaxVertexUniformBlocks), l.MaxUniformBuffersPerShaderStage),
		positive(gl.getParameterInt(glMaxFragmentUniformBlocks), l.MaxUniformBuffersPerShaderStage),
	)
	l.MaxDynamicUniformBuffersPerPipelineLayout = l.MaxUniformBuffersPerShaderStage
	l.MaxSampledTexturesPerShaderStage = maxTextureSlots
	l.MaxSamplersPerShaderStage = maxSamplers
	l.MaxUniformBufferBindingSize = uint64(positive(gl.getParameterInt(glMaxUniformBlockSize), uint32(l.MaxUniformBufferBindingSize)))
	colorAttachments := min(
		positive(gl.getParameterInt(glMaxColorAttachments), l.MaxColorAttachments),
		positive(gl.getParameterInt(glMaxDrawBuffers), l.MaxColorAttachments),
	)
	l.MaxColorAttachments = colorAttachments
	l.MaxColorAttachmentBytesPerSample = colorAttachments * 16
	l.MaxVertexAttributes = min(positive(gl.getParameterInt(glMaxVertexAttribs), l.MaxVertexAttributes), 16)
	l.MaxVertexBuffers = 16
	l.MaxBufferSize = 1<<31 - 1
	if v := gl.getParameterInt(glMaxVaryingComponents); v > 0 {
		l.MaxInterStageShaderVariables = uint32(v / 4)
	}
	return l
}

func (a *adapter) GetInfo() gpu.AdapterInfo { return a.info }
func (a *adapter) Features() gpu.Features   { return 0 }
func (a *adapter) Limits() gpu.Limits       { return a.limits }

func (a *adapter) GetTextureFormatFeatures(format gpu.TextureFormat) gpu.TextureFormatFeatures {
	desc, ok := describeTextureFormat(format)
	if !ok {
		return gpu.TextureFormatFeatures{}
	}
	usages := gpu.TextureUsagesCopySrc | gpu.TextureUsagesCopyDst | gpu.TextureUsagesTextureBinding
	if desc.filterable || desc.depth {
		usages |= gpu.TextureUsagesRenderAttachment
	}
	return gpu.TextureFormatFeatures{AllowedUsages: usages}
}

func (a *adapter) GetDownlevelCapabilities() gpu.DownlevelCapabilities {
	return gpu.DownlevelCapabilities{}
}

func limitsWithin(required, available gpu.Limits) error {
	checks := []struct {
		name                string
		required, available uint64
		alignment           bool
	}{
		{"max_texture_dimension_2d", uint64(required.MaxTextureDimension2D), uint64(available.MaxTextureDimension2D), false},
		{"max_uniform_buffer_binding_size", required.MaxUniformBufferBindingSize, available.MaxUniformBufferBindingSize, false},
		{"max_storage_buffers_per_shader_stage", uint64(required.MaxStorageBuffersPerShaderStage), uint64(available.MaxStorageBuffersPerShaderStage), false},
		{"min_uniform_buffer_offset_alignment", uint64(required.MinUniformBufferOffsetAlignment), uint64(available.MinUniformBufferOffsetAlignment), true},
	}
	for _, c := range checks {
		if c.alignment {
			if c.required < c.available {
				return fmt.Errorf("Limit '%s' value %d is better than allowed %d", c.name, c.required, c.available)
			}
			continue
		}
		if c.required > c.available {
			return fmt.Errorf("Limit '%s' value %d is better than allowed %d", c.name, c.required, c.available)
		}
	}
	return nil
}

func (a *adapter) RequestDevice(descriptor *gpu.DeviceDescriptor) (gpu.Device, gpu.Queue, error) {
	limits := a.limits
	if descriptor != nil {
		if missing := descriptor.RequiredFeatures &^ a.Features(); missing != 0 {
			return nil, nil, fmt.Errorf("Unsupported features were requested: %#x", uint64(missing))
		}
		if err := limitsWithin(descriptor.RequiredLimits, a.limits); err != nil {
			return nil, nil, err
		}
		limits = descriptor.RequiredLimits
	}
	d := &device{gl: a.gl, limits: limits}
	d.mainVAO = a.gl.createVertexArray()
	a.gl.bindVertexArray(d.mainVAO)
	return d, &queue{device: d}, nil
}

func (a *adapter) Release() {}

type errorScope struct {
	filter gpu.ErrorFilter
	err    *gpu.Error
}

type device struct {
	gl              glContext
	limits          gpu.Limits
	scopes          []errorScope
	uncaptured      func(err *gpu.Error)
	lostCallback    func(reason gpu.DeviceLostReason, message string)
	releaseLost     func()
	drawFramebuffer glObject
	readFramebuffer glObject
	mainVAO         glObject
	pendingMaps     []func()
	state           execState
	released        bool
}

func (d *device) Features() gpu.Features { return 0 }
func (d *device) Limits() gpu.Limits     { return d.limits }

func (d *device) SetDeviceLostCallback(callback func(reason gpu.DeviceLostReason, message string)) {
	d.lostCallback = callback
	if d.releaseLost == nil {
		d.releaseLost = d.gl.onContextLost(func() {
			if d.lostCallback != nil {
				d.lostCallback(gpu.DeviceLostReasonUnknown, "WebGL context lost")
			}
		})
	}
}

func (d *device) OnUncapturedError(handler func(err *gpu.Error)) { d.uncaptured = handler }

func (d *device) PushErrorScope(filter gpu.ErrorFilter) {
	d.scopes = append(d.scopes, errorScope{filter: filter})
}

func (d *device) PopErrorScope() *gpu.Error {
	if len(d.scopes) == 0 {
		return &gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Error scope stack is empty"}
	}
	top := d.scopes[len(d.scopes)-1]
	d.scopes = d.scopes[:len(d.scopes)-1]
	return top.err
}

func (d *device) report(filter gpu.ErrorFilter, message string) {
	for i := len(d.scopes) - 1; i >= 0; i-- {
		if d.scopes[i].filter == filter {
			if d.scopes[i].err == nil {
				d.scopes[i].err = &gpu.Error{Filter: filter, Message: message}
			}
			return
		}
	}
	if d.uncaptured != nil {
		d.uncaptured(&gpu.Error{Filter: filter, Message: message})
	}
}

func (d *device) CreateTexture(descriptor *gpu.TextureDescriptor) gpu.Texture {
	desc, ok := describeTextureFormat(descriptor.Format)
	t := &texture{
		gl:         d.gl,
		device:     d,
		width:      descriptor.Size.Width,
		height:     descriptor.Size.Height,
		format:     descriptor.Format,
		formatDesc: desc,
	}
	if !ok {
		d.report(gpu.ErrorFilterValidation, fmt.Sprintf("Texture format %s is not supported by the WebGL2 backend", descriptor.Format))
		return t
	}
	if descriptor.Dimension != 0 && descriptor.Dimension != gpu.TextureDimensionD2 || descriptor.Size.DepthOrArrayLayers > 1 {
		d.report(gpu.ErrorFilterValidation, "Only single-layer 2D textures are supported by the WebGL2 backend")
		return t
	}
	if t.width == 0 || t.height == 0 || t.width > d.limits.MaxTextureDimension2D || t.height > d.limits.MaxTextureDimension2D {
		d.report(gpu.ErrorFilterValidation, fmt.Sprintf("Dimension X value %d exceeds the limit of %d", max(t.width, t.height), d.limits.MaxTextureDimension2D))
		return t
	}
	levels := max(int(descriptor.MipLevelCount), 1)
	gl := d.gl
	t.raw = gl.createTexture()
	gl.bindTexture(glTexture2D, t.raw)
	if !desc.filterable {
		gl.texParameteri(glTexture2D, glTextureMinFilter, glNearest)
		gl.texParameteri(glTexture2D, glTextureMagFilter, glNearest)
	}
	gl.texStorage2D(glTexture2D, levels, desc.internal, int(t.width), int(t.height))
	gl.bindTexture(glTexture2D, 0)
	d.state.valid = false
	return t
}

func needsShadow(usage gpu.BufferUsages) bool {
	return usage&(gpu.BufferUsagesMapRead|gpu.BufferUsagesMapWrite) != 0
}

func needsGLBuffer(usage gpu.BufferUsages) bool {
	return usage&(gpu.BufferUsagesIndex|gpu.BufferUsagesVertex|gpu.BufferUsagesUniform|gpu.BufferUsagesStorage) != 0
}

func (d *device) CreateBuffer(descriptor *gpu.BufferDescriptor) gpu.Buffer {
	b := &buffer{device: d, size: descriptor.Size, usage: descriptor.Usage}
	if descriptor.Usage&gpu.BufferUsagesStorage != 0 {
		d.report(gpu.ErrorFilterValidation, "Storage buffers are not supported by the WebGL2 backend")
	}
	if needsShadow(descriptor.Usage) || descriptor.MappedAtCreation || !needsGLBuffer(descriptor.Usage) {
		b.shadow = make([]byte, descriptor.Size)
	}
	if needsGLBuffer(descriptor.Usage) {
		gl := d.gl
		b.raw = gl.createBuffer()
		gl.bindBuffer(glCopyWriteBuffer, b.raw)
		gl.bufferData(glCopyWriteBuffer, int(descriptor.Size), glDynamicDraw)
		gl.bindBuffer(glCopyWriteBuffer, 0)
	}
	b.mapped = descriptor.MappedAtCreation
	b.mapWrite = descriptor.MappedAtCreation
	return b
}

func (d *device) CreateSampler(descriptor *gpu.SamplerDescriptor) gpu.Sampler {
	gl := d.gl
	raw := gl.createSampler()
	minFilter, magFilter := mapFilterModes(descriptor.MinFilter, descriptor.MagFilter, descriptor.MipmapFilter)
	gl.samplerParameteri(raw, glTextureMinFilter, int(minFilter))
	gl.samplerParameteri(raw, glTextureMagFilter, int(magFilter))
	gl.samplerParameteri(raw, glTextureWrapS, int(mapAddressMode(descriptor.AddressModeU)))
	gl.samplerParameteri(raw, glTextureWrapT, int(mapAddressMode(descriptor.AddressModeV)))
	gl.samplerParameteri(raw, glTextureWrapR, int(mapAddressMode(descriptor.AddressModeW)))
	lodMax := descriptor.LodMaxClamp
	if lodMax == 0 {
		lodMax = 32
	}
	gl.samplerParameterf(raw, glTextureMinLod, descriptor.LodMinClamp)
	gl.samplerParameterf(raw, glTextureMaxLod, lodMax)
	if descriptor.Compare != gpu.CompareFunctionUndefined {
		gl.samplerParameteri(raw, glTextureCompareMode, glCompareRefToTexture)
		gl.samplerParameteri(raw, glTextureCompareFunc, int(mapCompareFunc(descriptor.Compare)))
	}
	return &sampler{gl: gl, raw: raw}
}

func (d *device) CreateShaderModule(descriptor *gpu.ShaderModuleDescriptor) gpu.ShaderModule {
	sum := sha256.Sum256([]byte(descriptor.WGSL))
	return &shaderModule{label: descriptor.Label, translated: hex.EncodeToString(sum[:]) == webglsl.WebglWGSLSHA256}
}

func (d *device) CreateBindGroupLayout(descriptor *gpu.BindGroupLayoutDescriptor) gpu.BindGroupLayout {
	return &bindGroupLayout{entries: append([]gpu.BindGroupLayoutEntry(nil), descriptor.Entries...)}
}

func (d *device) CreateBindGroup(descriptor *gpu.BindGroupDescriptor) gpu.BindGroup {
	layout, _ := descriptor.Layout.(*bindGroupLayout)
	group := &bindGroup{layout: layout}
	if layout == nil {
		d.report(gpu.ErrorFilterValidation, "Bind group layout is invalid")
		return group
	}
	for _, entry := range layout.entries {
		found := false
		for _, e := range descriptor.Entries {
			if e.Binding == entry.Binding {
				group.contents = append(group.contents, e.Resource)
				found = true
				break
			}
		}
		if !found {
			d.report(gpu.ErrorFilterValidation, fmt.Sprintf("Number of bindings in bind group descriptor (%d) does not match the number of bindings defined in the bind group layout (%d)", len(descriptor.Entries), len(layout.entries)))
			group.contents = append(group.contents, gpu.BindingResource{})
		}
	}
	return group
}

func (d *device) CreatePipelineLayout(descriptor *gpu.PipelineLayoutDescriptor) gpu.PipelineLayout {
	layout, err := createPipelineLayout(descriptor.BindGroupLayouts)
	if err != nil {
		d.report(gpu.ErrorFilterValidation, err.Error())
	}
	return layout
}

func (d *device) CreateRenderPipeline(descriptor *gpu.RenderPipelineDescriptor) gpu.RenderPipeline {
	p, err := d.createRenderPipeline(descriptor)
	if err != nil {
		d.report(gpu.ErrorFilterValidation, err.Error())
	}
	return p
}

func (d *device) CreateCommandEncoder(*gpu.CommandEncoderDescriptor) gpu.CommandEncoder {
	return &commandEncoder{device: d}
}

func (d *device) Poll(bool) error {
	pending := d.pendingMaps
	d.pendingMaps = nil
	for _, f := range pending {
		f()
	}
	if d.gl.isContextLost() {
		return errors.New("Parent device is lost")
	}
	return nil
}

func (d *device) Release() {
	if d.released {
		return
	}
	d.released = true
	if d.releaseLost != nil {
		d.releaseLost()
		d.releaseLost = nil
	}
	if d.drawFramebuffer != 0 {
		d.gl.deleteFramebuffer(d.drawFramebuffer)
		d.drawFramebuffer = 0
	}
	if d.readFramebuffer != 0 {
		d.gl.deleteFramebuffer(d.readFramebuffer)
		d.readFramebuffer = 0
	}
	if d.mainVAO != 0 {
		d.gl.deleteVertexArray(d.mainVAO)
		d.mainVAO = 0
	}
}

type queue struct {
	device *device
}

func (q *queue) WriteBuffer(b gpu.Buffer, offset uint64, data []byte) {
	buf, ok := b.(*buffer)
	if !ok || buf.destroyed {
		q.device.report(gpu.ErrorFilterValidation, "Buffer is invalid")
		return
	}
	if offset+uint64(len(data)) > buf.size {
		q.device.report(gpu.ErrorFilterValidation, fmt.Sprintf("Copy of %d..%d would end up overrunning the bounds of the Destination buffer of size %d", offset, offset+uint64(len(data)), buf.size))
		return
	}
	if buf.shadow != nil {
		copy(buf.shadow[offset:], data)
	}
	if buf.raw != 0 {
		gl := q.device.gl
		gl.bindBuffer(glCopyWriteBuffer, buf.raw)
		gl.bufferSubData(glCopyWriteBuffer, int(offset), data)
		gl.bindBuffer(glCopyWriteBuffer, 0)
	}
}

func (q *queue) WriteTexture(target gpu.TexelCopyTextureInfo, data []byte, layout gpu.TexelCopyBufferLayout, size gpu.Extent3d) {
	t, ok := target.Texture.(*texture)
	if !ok || t.raw == 0 || t.destroyed {
		q.device.report(gpu.ErrorFilterValidation, "Texture is invalid")
		return
	}
	bpp := t.formatDesc.bytesPerTexel
	rowBytes := size.Width * bpp
	stride := layout.BytesPerRow
	if stride == 0 {
		stride = rowBytes
	}
	if size.Width == 0 || size.Height == 0 {
		return
	}
	needed := uint64(stride)*uint64(size.Height-1) + uint64(rowBytes)
	if layout.Offset+needed > uint64(len(data)) || target.Origin.X+size.Width > t.width || target.Origin.Y+size.Height > t.height {
		q.device.report(gpu.ErrorFilterValidation, "Copy would overrun the texture or the source data")
		return
	}
	src := data[layout.Offset : layout.Offset+needed]
	if t.formatDesc.swizzleBgra {
		src = swizzleRB(src)
	}
	gl := q.device.gl
	gl.bindTexture(glTexture2D, t.raw)
	gl.pixelStorei(glUnpackAlignment, 1)
	if stride != rowBytes {
		gl.pixelStorei(glUnpackRowLength, int(stride/bpp))
	}
	gl.texSubImage2D(glTexture2D, int(target.MipLevel), int(target.Origin.X), int(target.Origin.Y), int(size.Width), int(size.Height), t.formatDesc.external, t.formatDesc.dataType, src)
	if stride != rowBytes {
		gl.pixelStorei(glUnpackRowLength, 0)
	}
	gl.bindTexture(glTexture2D, 0)
	q.device.state.valid = false
}

func (q *queue) Submit(commands ...gpu.CommandBuffer) gpu.SubmissionIndex {
	for _, c := range commands {
		cb, ok := c.(*commandBuffer)
		if !ok {
			continue
		}
		q.device.execute(cb.commands)
	}
	return 0
}

func (q *queue) Release() {}

type texture struct {
	gl           glContext
	device       *device
	raw          glObject
	width        uint32
	height       uint32
	format       gpu.TextureFormat
	formatDesc   textureFormatDesc
	surfaceOwned bool
	destroyed    bool
}

func (t *texture) CreateView(*gpu.TextureViewDescriptor) gpu.TextureView {
	return &textureView{texture: t}
}

func (t *texture) Width() uint32             { return t.width }
func (t *texture) Height() uint32            { return t.height }
func (t *texture) Format() gpu.TextureFormat { return t.format }

func (t *texture) Destroy() {
	if t.destroyed || t.surfaceOwned {
		return
	}
	t.destroyed = true
	if t.raw != 0 {
		t.gl.deleteTexture(t.raw)
	}
}

func (t *texture) Release() { t.Destroy() }

type textureView struct {
	texture *texture
}

func (v *textureView) Release() {}

type buffer struct {
	device    *device
	raw       glObject
	size      uint64
	usage     gpu.BufferUsages
	shadow    []byte
	mapped    bool
	mapWrite  bool
	destroyed bool
}

func (b *buffer) Size() uint64 { return b.size }

func (b *buffer) MapAsync(mode gpu.MapMode, offset, size uint64, callback func(err error)) {
	if b.shadow == nil || b.destroyed {
		b.device.pendingMaps = append(b.device.pendingMaps, func() { callback(errors.New("Buffer is invalid or not mappable")) })
		return
	}
	if offset+size > b.size {
		b.device.pendingMaps = append(b.device.pendingMaps, func() { callback(errors.New("Buffer map range out of bounds")) })
		return
	}
	b.device.pendingMaps = append(b.device.pendingMaps, func() {
		b.mapped = true
		b.mapWrite = mode&gpu.MapModeWrite != 0
		callback(nil)
	})
}

func (b *buffer) GetMappedRange(offset, size uint64) []byte {
	if !b.mapped || b.shadow == nil || offset+size > uint64(len(b.shadow)) {
		return nil
	}
	return b.shadow[offset : offset+size]
}

func (b *buffer) Unmap() {
	if !b.mapped {
		return
	}
	b.mapped = false
	if b.mapWrite && b.raw != 0 {
		gl := b.device.gl
		gl.bindBuffer(glCopyWriteBuffer, b.raw)
		gl.bufferSubData(glCopyWriteBuffer, 0, b.shadow)
		gl.bindBuffer(glCopyWriteBuffer, 0)
	}
	b.mapWrite = false
}

func (b *buffer) Destroy() {
	if b.destroyed {
		return
	}
	b.destroyed = true
	if b.raw != 0 {
		b.device.gl.deleteBuffer(b.raw)
		b.raw = 0
	}
	b.shadow = nil
}

func (b *buffer) Release() { b.Destroy() }

type sampler struct {
	gl       glContext
	raw      glObject
	released bool
}

func (s *sampler) Release() {
	if s.released {
		return
	}
	s.released = true
	s.gl.deleteSampler(s.raw)
}

type shaderModule struct {
	label      string
	translated bool
}

func (m *shaderModule) Release() {}

type bindGroupLayout struct {
	entries []gpu.BindGroupLayoutEntry
}

func (l *bindGroupLayout) Release() {}

type bindGroup struct {
	layout   *bindGroupLayout
	contents []gpu.BindingResource
}

func (g *bindGroup) Release() {}
