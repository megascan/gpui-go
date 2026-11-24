//go:build linux || js

// Package webgpu implements the gpu facade over github.com/oliverbestmann/webgpu (wgpu-native).
package webgpu

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// SupportsTransientAttachment reports whether wgpu-native accepts the TRANSIENT_ATTACHMENT
// texture usage; v29 aborts the process converting it.
const SupportsTransientAttachment = false

// CreateInstance creates a wgpu-native instance for the given backends.
func CreateInstance(descriptor gpu.InstanceDescriptor) gpu.Instance {
	return &instance{raw: wgpu.CreateInstance(&wgpu.InstanceDescriptor{Backends: instanceBackends(descriptor.Backends)})}
}

func instanceBackends(backends gpu.Backends) wgpu.InstanceBackend {
	var out wgpu.InstanceBackend
	if backends&gpu.BackendsVulkan != 0 {
		out |= 1 << 0
	}
	if backends&gpu.BackendsGl != 0 {
		out |= 1 << 1
	}
	if backends&gpu.BackendsMetal != 0 {
		out |= 1 << 2
	}
	if backends&gpu.BackendsDx12 != 0 {
		out |= 1 << 3
	}
	if backends&gpu.BackendsBrowserWebGpu != 0 {
		out |= 1 << 5
	}
	return out
}

type instance struct {
	raw *wgpu.Instance
}

func (i *instance) EnumerateAdapters(backends gpu.Backends) []gpu.Adapter {
	raw := i.raw.EnumerateAdapters(&wgpu.InstanceEnumerateAdapterOptons{Backends: instanceBackends(backends)})
	adapters := make([]gpu.Adapter, 0, len(raw))
	for _, a := range raw {
		adapters = append(adapters, &adapter{raw: a})
	}
	return adapters
}

func (i *instance) RequestAdapter(options *gpu.RequestAdapterOptions) (gpu.Adapter, error) {
	opts := &wgpu.RequestAdapterOptions{}
	if options != nil {
		switch options.PowerPreference {
		case gpu.PowerPreferenceLowPower:
			opts.PowerPreference = wgpu.PowerPreferenceLowPower
		case gpu.PowerPreferenceHighPerformance:
			opts.PowerPreference = wgpu.PowerPreferenceHighPerformance
		}
		if s, ok := options.CompatibleSurface.(*surface); ok && s != nil {
			opts.CompatibleSurface = s.raw
		}
		opts.ForceFallbackAdapter = options.ForceFallbackAdapter
	}
	a, err := i.raw.RequestAdapter(opts)
	if err != nil {
		return nil, err
	}
	return &adapter{raw: a}, nil
}

func (i *instance) CreateSurface(target gpu.SurfaceTarget) (gpu.Surface, error) {
	desc, err := surfaceDescriptor(target)
	if err != nil {
		return nil, err
	}
	return i.createSurface(desc)
}

func (i *instance) createSurface(desc *wgpu.SurfaceDescriptor) (gpu.Surface, error) {
	var created *wgpu.Surface
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("%v", r)
			}
		}()
		created = i.raw.CreateSurface(desc)
	}()
	if err != nil {
		return nil, err
	}
	return &surface{raw: created}, nil
}

func (i *instance) Release() {
	i.raw.Release()
}

type adapter struct {
	raw *wgpu.Adapter
}

func (a *adapter) GetInfo() gpu.AdapterInfo {
	info := a.raw.GetInfo()
	return gpu.AdapterInfo{
		Name:       info.Device,
		Vendor:     info.VendorId,
		Device:     info.DeviceId,
		DeviceType: deviceType(info.AdapterType),
		Driver:     info.Vendor,
		DriverInfo: info.Description,
		Backend:    adapterBackend(info),
	}
}

func deviceType(t wgpu.AdapterType) gpu.DeviceType {
	switch t {
	case wgpu.AdapterTypeDiscreteGPU:
		return gpu.DeviceTypeDiscreteGpu
	case wgpu.AdapterTypeIntegratedGPU:
		return gpu.DeviceTypeIntegratedGpu
	case wgpu.AdapterTypeCPU:
		return gpu.DeviceTypeCpu
	}
	return gpu.DeviceTypeOther
}

func backend(t wgpu.BackendType) gpu.Backend {
	switch t {
	case wgpu.BackendTypeVulkan:
		return gpu.BackendVulkan
	case wgpu.BackendTypeMetal:
		return gpu.BackendMetal
	case wgpu.BackendTypeD3D12:
		return gpu.BackendDx12
	case wgpu.BackendTypeOpenGL, wgpu.BackendTypeOpenGLES:
		return gpu.BackendGl
	case wgpu.BackendTypeWebGPU:
		return gpu.BackendBrowserWebGpu
	}
	return gpu.BackendNoop
}

func (a *adapter) Features() gpu.Features {
	var features gpu.Features
	if a.raw.HasFeature(wgpu.FeatureNameDualSourceBlending) {
		features |= gpu.FeaturesDualSourceBlending
	}
	return features
}

func (a *adapter) Limits() gpu.Limits {
	return fromLimits(a.raw.GetLimits())
}

func (a *adapter) GetTextureFormatFeatures(format gpu.TextureFormat) gpu.TextureFormatFeatures {
	base := gpu.TextureUsagesCopySrc | gpu.TextureUsagesCopyDst | gpu.TextureUsagesTextureBinding
	switch format {
	case gpu.TextureFormatR8Unorm, gpu.TextureFormatBgra8Unorm, gpu.TextureFormatBgra8UnormSrgb, gpu.TextureFormatRgba8UnormSrgb:
		return gpu.TextureFormatFeatures{AllowedUsages: base | gpu.TextureUsagesRenderAttachment}
	case gpu.TextureFormatRgba8Unorm, gpu.TextureFormatRgba32Uint:
		return gpu.TextureFormatFeatures{AllowedUsages: base | gpu.TextureUsagesRenderAttachment | gpu.TextureUsagesStorageBinding}
	case gpu.TextureFormatDepth32Float:
		return gpu.TextureFormatFeatures{AllowedUsages: gpu.TextureUsagesCopySrc | gpu.TextureUsagesTextureBinding | gpu.TextureUsagesRenderAttachment}
	}
	return gpu.TextureFormatFeatures{}
}

func (a *adapter) GetDownlevelCapabilities() gpu.DownlevelCapabilities {
	flags := gpu.DownlevelFlagsVertexStorage | gpu.DownlevelFlagsFragmentStorage
	if adapterBackend(a.raw.GetInfo()) == gpu.BackendGl && a.raw.GetLimits().MaxStorageBuffersPerShaderStage == 0 {
		flags = 0
	}
	return gpu.DownlevelCapabilities{Flags: flags}
}

func (a *adapter) RequestDevice(descriptor *gpu.DeviceDescriptor) (gpu.Device, gpu.Queue, error) {
	d := &device{adapter: a}
	desc := &wgpu.DeviceDescriptor{
		DeviceLostCallback: d.lost,
	}
	if descriptor != nil {
		desc.Label = descriptor.Label
		if descriptor.RequiredFeatures.Contains(gpu.FeaturesDualSourceBlending) {
			desc.RequiredFeatures = append(desc.RequiredFeatures, wgpu.FeatureNameDualSourceBlending)
		}
		limits := toLimits(descriptor.RequiredLimits)
		desc.RequiredLimits = &limits
	}
	raw, err := a.raw.RequestDevice(desc)
	if err != nil {
		return nil, nil, err
	}
	d.raw = raw
	q := &queue{device: d, raw: raw.GetQueue()}
	return d, q, nil
}

func (a *adapter) Release() {
	a.raw.Release()
}

func fromLimits(l wgpu.Limits) gpu.Limits {
	return gpu.Limits{
		MaxTextureDimension1D:                     l.MaxTextureDimension1D,
		MaxTextureDimension2D:                     l.MaxTextureDimension2D,
		MaxTextureDimension3D:                     l.MaxTextureDimension3D,
		MaxTextureArrayLayers:                     l.MaxTextureArrayLayers,
		MaxBindGroups:                             l.MaxBindGroups,
		MaxBindingsPerBindGroup:                   l.MaxBindingsPerBindGroup,
		MaxDynamicUniformBuffersPerPipelineLayout: l.MaxDynamicUniformBuffersPerPipelineLayout,
		MaxDynamicStorageBuffersPerPipelineLayout: l.MaxDynamicStorageBuffersPerPipelineLayout,
		MaxSampledTexturesPerShaderStage:          l.MaxSampledTexturesPerShaderStage,
		MaxSamplersPerShaderStage:                 l.MaxSamplersPerShaderStage,
		MaxStorageBuffersPerShaderStage:           l.MaxStorageBuffersPerShaderStage,
		MaxStorageTexturesPerShaderStage:          l.MaxStorageTexturesPerShaderStage,
		MaxUniformBuffersPerShaderStage:           l.MaxUniformBuffersPerShaderStage,
		MaxUniformBufferBindingSize:               l.MaxUniformBufferBindingSize,
		MaxStorageBufferBindingSize:               l.MaxStorageBufferBindingSize,
		MinUniformBufferOffsetAlignment:           l.MinUniformBufferOffsetAlignment,
		MinStorageBufferOffsetAlignment:           l.MinStorageBufferOffsetAlignment,
		MaxVertexBuffers:                          l.MaxVertexBuffers,
		MaxBufferSize:                             l.MaxBufferSize,
		MaxVertexAttributes:                       l.MaxVertexAttributes,
		MaxVertexBufferArrayStride:                l.MaxVertexBufferArrayStride,
		MaxInterStageShaderVariables:              l.MaxInterStageShaderVariables,
		MaxColorAttachments:                       l.MaxColorAttachments,
		MaxColorAttachmentBytesPerSample:          l.MaxColorAttachmentBytesPerSample,
		MaxComputeWorkgroupStorageSize:            l.MaxComputeWorkgroupStorageSize,
		MaxComputeInvocationsPerWorkgroup:         l.MaxComputeInvocationsPerWorkgroup,
		MaxComputeWorkgroupSizeX:                  l.MaxComputeWorkgroupSizeX,
		MaxComputeWorkgroupSizeY:                  l.MaxComputeWorkgroupSizeY,
		MaxComputeWorkgroupSizeZ:                  l.MaxComputeWorkgroupSizeZ,
		MaxComputeWorkgroupsPerDimension:          l.MaxComputeWorkgroupsPerDimension,
		MaxImmediateSize:                          l.MaxImmediateSize,
		MaxNonSamplerBindings:                     l.MaxNonSamplerBindings,
	}
}

func toLimits(l gpu.Limits) wgpu.Limits {
	out := wgpu.DefaultLimits()
	out.MaxTextureDimension1D = l.MaxTextureDimension1D
	out.MaxTextureDimension2D = l.MaxTextureDimension2D
	out.MaxTextureDimension3D = l.MaxTextureDimension3D
	out.MaxTextureArrayLayers = l.MaxTextureArrayLayers
	out.MaxBindGroups = l.MaxBindGroups
	out.MaxBindingsPerBindGroup = l.MaxBindingsPerBindGroup
	out.MaxDynamicUniformBuffersPerPipelineLayout = l.MaxDynamicUniformBuffersPerPipelineLayout
	out.MaxDynamicStorageBuffersPerPipelineLayout = l.MaxDynamicStorageBuffersPerPipelineLayout
	out.MaxSampledTexturesPerShaderStage = l.MaxSampledTexturesPerShaderStage
	out.MaxSamplersPerShaderStage = l.MaxSamplersPerShaderStage
	out.MaxStorageBuffersPerShaderStage = l.MaxStorageBuffersPerShaderStage
	out.MaxStorageTexturesPerShaderStage = l.MaxStorageTexturesPerShaderStage
	out.MaxUniformBuffersPerShaderStage = l.MaxUniformBuffersPerShaderStage
	out.MaxUniformBufferBindingSize = l.MaxUniformBufferBindingSize
	out.MaxStorageBufferBindingSize = l.MaxStorageBufferBindingSize
	out.MinUniformBufferOffsetAlignment = l.MinUniformBufferOffsetAlignment
	out.MinStorageBufferOffsetAlignment = l.MinStorageBufferOffsetAlignment
	out.MaxVertexBuffers = l.MaxVertexBuffers
	out.MaxBufferSize = l.MaxBufferSize
	out.MaxVertexAttributes = l.MaxVertexAttributes
	out.MaxVertexBufferArrayStride = l.MaxVertexBufferArrayStride
	out.MaxInterStageShaderVariables = l.MaxInterStageShaderVariables
	out.MaxColorAttachments = l.MaxColorAttachments
	out.MaxColorAttachmentBytesPerSample = l.MaxColorAttachmentBytesPerSample
	out.MaxComputeWorkgroupStorageSize = l.MaxComputeWorkgroupStorageSize
	out.MaxComputeInvocationsPerWorkgroup = l.MaxComputeInvocationsPerWorkgroup
	out.MaxComputeWorkgroupSizeX = l.MaxComputeWorkgroupSizeX
	out.MaxComputeWorkgroupSizeY = l.MaxComputeWorkgroupSizeY
	out.MaxComputeWorkgroupSizeZ = l.MaxComputeWorkgroupSizeZ
	out.MaxComputeWorkgroupsPerDimension = l.MaxComputeWorkgroupsPerDimension
	out.MaxImmediateSize = l.MaxImmediateSize
	out.MaxNonSamplerBindings = l.MaxNonSamplerBindings
	out.MaxBindingArrayElementsPerShaderStage = 0
	out.MaxBindingArraySamplerElementsPerShaderStage = 0
	return out
}

type errorScope struct {
	filter gpu.ErrorFilter
	err    *gpu.Error
}

type device struct {
	raw     *wgpu.Device
	adapter *adapter

	mu         sync.Mutex
	lostFn     func(reason gpu.DeviceLostReason, message string)
	uncaptured func(err *gpu.Error)
	scopes     []errorScope
}

func (d *device) lost(reason wgpu.DeviceLostReason, message string) {
	d.mu.Lock()
	fn := d.lostFn
	d.mu.Unlock()
	if fn == nil {
		return
	}
	r := gpu.DeviceLostReasonUnknown
	if reason == wgpu.DeviceLostReasonDestroyed || reason == wgpu.DeviceLostReasonCallbackCancelled {
		r = gpu.DeviceLostReasonDestroyed
	}
	fn(r, message)
}

func (d *device) report(err *gpu.Error) {
	if err == nil {
		return
	}
	d.mu.Lock()
	for i := len(d.scopes) - 1; i >= 0; i-- {
		if d.scopes[i].filter == err.Filter {
			if d.scopes[i].err == nil {
				d.scopes[i].err = err
			}
			d.mu.Unlock()
			return
		}
	}
	handler := d.uncaptured
	d.mu.Unlock()
	if handler == nil {
		panic("wgpu error: " + err.Message)
	}
	handler(err)
}

func (d *device) reportErr(err error) {
	if err == nil {
		return
	}
	d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: err.Error()})
}

func (d *device) Features() gpu.Features {
	var features gpu.Features
	if d.raw.HasFeature(wgpu.FeatureNameDualSourceBlending) {
		features |= gpu.FeaturesDualSourceBlending
	}
	return features
}

func (d *device) Limits() gpu.Limits {
	return fromLimits(d.raw.GetLimits())
}

func (d *device) SetDeviceLostCallback(callback func(reason gpu.DeviceLostReason, message string)) {
	d.mu.Lock()
	d.lostFn = callback
	d.mu.Unlock()
}

func (d *device) OnUncapturedError(handler func(err *gpu.Error)) {
	d.mu.Lock()
	d.uncaptured = handler
	d.mu.Unlock()
}

func (d *device) PushErrorScope(filter gpu.ErrorFilter) {
	d.mu.Lock()
	d.scopes = append(d.scopes, errorScope{filter: filter})
	d.mu.Unlock()
}

func (d *device) PopErrorScope() *gpu.Error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.scopes) == 0 {
		panic("wgpu: PopErrorScope without a matching PushErrorScope")
	}
	scope := d.scopes[len(d.scopes)-1]
	d.scopes = d.scopes[:len(d.scopes)-1]
	return scope.err
}

func (d *device) CreateTexture(descriptor *gpu.TextureDescriptor) gpu.Texture {
	if descriptor.Usage&gpu.TextureUsagesTransientAttachment != 0 && !SupportsTransientAttachment {
		d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "TRANSIENT_ATTACHMENT is not supported by wgpu-native v29"})
		return &texture{device: d, width: descriptor.Size.Width, height: descriptor.Size.Height, format: descriptor.Format}
	}
	mips := descriptor.MipLevelCount
	if mips == 0 {
		mips = 1
	}
	samples := descriptor.SampleCount
	if samples == 0 {
		samples = 1
	}
	depth := descriptor.Size.DepthOrArrayLayers
	if depth == 0 {
		depth = 1
	}
	viewFormats := make([]wgpu.TextureFormat, len(descriptor.ViewFormats))
	for i, f := range descriptor.ViewFormats {
		viewFormats[i] = wgpu.TextureFormat(f)
	}
	raw, err := d.raw.TryCreateTexture(&wgpu.TextureDescriptor{
		Label:         descriptor.Label,
		Usage:         wgpu.TextureUsage(descriptor.Usage),
		Dimension:     wgpu.TextureDimension(descriptor.Dimension),
		Size:          wgpu.Extent3D{Width: descriptor.Size.Width, Height: descriptor.Size.Height, DepthOrArrayLayers: depth},
		Format:        wgpu.TextureFormat(descriptor.Format),
		MipLevelCount: mips,
		SampleCount:   samples,
		ViewFormats:   viewFormats,
	})
	d.reportErr(err)
	return &texture{device: d, raw: raw, width: descriptor.Size.Width, height: descriptor.Size.Height, format: descriptor.Format}
}

func (d *device) CreateBuffer(descriptor *gpu.BufferDescriptor) gpu.Buffer {
	raw, err := d.raw.TryCreateBuffer(&wgpu.BufferDescriptor{
		Label:            descriptor.Label,
		Usage:            wgpu.BufferUsage(descriptor.Usage),
		Size:             descriptor.Size,
		MappedAtCreation: descriptor.MappedAtCreation,
	})
	d.reportErr(err)
	return &buffer{device: d, raw: raw, size: descriptor.Size}
}

func (d *device) CreateSampler(descriptor *gpu.SamplerDescriptor) gpu.Sampler {
	lodMax := descriptor.LodMaxClamp
	if lodMax == 0 {
		lodMax = 32
	}
	raw, err := d.raw.TryCreateSampler(&wgpu.SamplerDescriptor{
		Label:         descriptor.Label,
		AddressModeU:  addressMode(descriptor.AddressModeU),
		AddressModeV:  addressMode(descriptor.AddressModeV),
		AddressModeW:  addressMode(descriptor.AddressModeW),
		MagFilter:     filterMode(descriptor.MagFilter),
		MinFilter:     filterMode(descriptor.MinFilter),
		MipmapFilter:  wgpu.MipmapFilterMode(filterMode(descriptor.MipmapFilter)),
		LodMinClamp:   descriptor.LodMinClamp,
		LodMaxClamp:   lodMax,
		Compare:       wgpu.CompareFunction(descriptor.Compare),
		MaxAnisotropy: 1,
	})
	d.reportErr(err)
	return &sampler{raw: raw}
}

func addressMode(m gpu.AddressMode) wgpu.AddressMode {
	if m == 0 {
		return wgpu.AddressModeClampToEdge
	}
	return wgpu.AddressMode(m)
}

func filterMode(m gpu.FilterMode) wgpu.FilterMode {
	if m == 0 {
		return wgpu.FilterModeNearest
	}
	return wgpu.FilterMode(m)
}

func (d *device) CreateShaderModule(descriptor *gpu.ShaderModuleDescriptor) gpu.ShaderModule {
	raw, err := d.raw.TryCreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label:      descriptor.Label,
		WGSLSource: &wgpu.ShaderSourceWGSL{Code: descriptor.WGSL},
	})
	d.reportErr(err)
	return &shaderModule{raw: raw}
}

func (d *device) CreateBindGroupLayout(descriptor *gpu.BindGroupLayoutDescriptor) gpu.BindGroupLayout {
	entries := make([]wgpu.BindGroupLayoutEntry, len(descriptor.Entries))
	for i, e := range descriptor.Entries {
		entry := wgpu.BindGroupLayoutEntry{Binding: e.Binding, Visibility: wgpu.ShaderStage(e.Visibility)}
		switch {
		case e.Type.Buffer != nil:
			entry.Buffer = wgpu.BufferBindingLayout{
				Type:             wgpu.BufferBindingType(e.Type.Buffer.Type),
				HasDynamicOffset: e.Type.Buffer.HasDynamicOffset,
				MinBindingSize:   e.Type.Buffer.MinBindingSize,
			}
		case e.Type.Sampler != nil:
			entry.Sampler = wgpu.SamplerBindingLayout{Type: wgpu.SamplerBindingType(e.Type.Sampler.Type)}
		case e.Type.Texture != nil:
			entry.Texture = wgpu.TextureBindingLayout{
				SampleType:    wgpu.TextureSampleType(e.Type.Texture.SampleType),
				ViewDimension: wgpu.TextureViewDimension(e.Type.Texture.ViewDimension),
				Multisampled:  e.Type.Texture.Multisampled,
			}
		}
		entries[i] = entry
	}
	raw, err := d.raw.TryCreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{Label: descriptor.Label, Entries: entries})
	d.reportErr(err)
	return &bindGroupLayout{raw: raw}
}

func (d *device) CreateBindGroup(descriptor *gpu.BindGroupDescriptor) gpu.BindGroup {
	layout, _ := descriptor.Layout.(*bindGroupLayout)
	if layout == nil || layout.raw == nil {
		d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "BindGroupLayout is invalid"})
		return &bindGroup{}
	}
	entries := make([]wgpu.BindGroupEntry, len(descriptor.Entries))
	for i, e := range descriptor.Entries {
		entry := wgpu.BindGroupEntry{Binding: e.Binding}
		switch {
		case e.Resource.Buffer != nil:
			b, _ := e.Resource.Buffer.(*buffer)
			if b == nil || b.raw == nil {
				d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Buffer is invalid"})
				return &bindGroup{}
			}
			entry.Buffer = b.raw
			entry.Offset = e.Resource.Offset
			entry.Size = e.Resource.Size
			if entry.Size == 0 {
				entry.Size = wgpu.WholeSize
			}
		case e.Resource.Sampler != nil:
			s, _ := e.Resource.Sampler.(*sampler)
			if s == nil || s.raw == nil {
				d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Sampler is invalid"})
				return &bindGroup{}
			}
			entry.Sampler = s.raw
		case e.Resource.TextureView != nil:
			v, _ := e.Resource.TextureView.(*textureView)
			if v == nil || v.raw == nil {
				d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "TextureView is invalid"})
				return &bindGroup{}
			}
			entry.TextureView = v.raw
		}
		entries[i] = entry
	}
	raw, err := d.raw.TryCreateBindGroup(&wgpu.BindGroupDescriptor{Label: descriptor.Label, Layout: layout.raw, Entries: entries})
	d.reportErr(err)
	return &bindGroup{raw: raw}
}

func (d *device) CreatePipelineLayout(descriptor *gpu.PipelineLayoutDescriptor) gpu.PipelineLayout {
	layouts := make([]*wgpu.BindGroupLayout, len(descriptor.BindGroupLayouts))
	for i, l := range descriptor.BindGroupLayouts {
		bgl, _ := l.(*bindGroupLayout)
		if bgl == nil || bgl.raw == nil {
			d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "BindGroupLayout is invalid"})
			return &pipelineLayout{}
		}
		layouts[i] = bgl.raw
	}
	raw, err := d.raw.TryCreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{Label: descriptor.Label, BindGroupLayouts: layouts})
	d.reportErr(err)
	return &pipelineLayout{raw: raw}
}

func (d *device) CreateRenderPipeline(descriptor *gpu.RenderPipelineDescriptor) gpu.RenderPipeline {
	invalid := func(what string) gpu.RenderPipeline {
		d.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: what + " is invalid"})
		return &renderPipeline{}
	}
	var layout *wgpu.PipelineLayout
	if descriptor.Layout != nil {
		pl, _ := descriptor.Layout.(*pipelineLayout)
		if pl == nil || pl.raw == nil {
			return invalid("PipelineLayout")
		}
		layout = pl.raw
	}
	vs, _ := descriptor.Vertex.Module.(*shaderModule)
	if vs == nil || vs.raw == nil {
		return invalid("ShaderModule")
	}
	desc := &wgpu.RenderPipelineDescriptor{
		Label:  descriptor.Label,
		Layout: layout,
		Vertex: wgpu.VertexState{Module: vs.raw, EntryPoint: descriptor.Vertex.EntryPoint},
		Primitive: wgpu.PrimitiveState{
			Topology:  wgpu.PrimitiveTopology(descriptor.Primitive.Topology),
			FrontFace: frontFace(descriptor.Primitive.FrontFace),
			CullMode:  cullMode(descriptor.Primitive.CullMode),
		},
		Multisample: wgpu.MultisampleState{
			Count:                  max(descriptor.Multisample.Count, 1),
			Mask:                   descriptor.Multisample.Mask,
			AlphaToCoverageEnabled: descriptor.Multisample.AlphaToCoverageEnabled,
		},
	}
	if desc.Multisample.Mask == 0 {
		desc.Multisample.Mask = ^uint32(0)
	}
	if ds := descriptor.DepthStencil; ds != nil {
		write := wgpu.OptionalBoolFalse
		if ds.DepthWriteEnabled {
			write = wgpu.OptionalBoolTrue
		}
		desc.DepthStencil = &wgpu.DepthStencilState{
			Format:              wgpu.TextureFormat(ds.Format),
			DepthWriteEnabled:   write,
			DepthCompare:        wgpu.CompareFunction(ds.DepthCompare),
			StencilFront:        wgpu.StencilFaceState{Compare: wgpu.CompareFunctionAlways, FailOp: wgpu.StencilOperationKeep, DepthFailOp: wgpu.StencilOperationKeep, PassOp: wgpu.StencilOperationKeep},
			StencilBack:         wgpu.StencilFaceState{Compare: wgpu.CompareFunctionAlways, FailOp: wgpu.StencilOperationKeep, DepthFailOp: wgpu.StencilOperationKeep, PassOp: wgpu.StencilOperationKeep},
			DepthBias:           ds.DepthBias,
			DepthBiasSlopeScale: ds.DepthBiasSlopeScale,
			DepthBiasClamp:      ds.DepthBiasClamp,
		}
	}
	if fs := descriptor.Fragment; fs != nil {
		module, _ := fs.Module.(*shaderModule)
		if module == nil || module.raw == nil {
			return invalid("ShaderModule")
		}
		targets := make([]wgpu.ColorTargetState, len(fs.Targets))
		for i, t := range fs.Targets {
			target := wgpu.ColorTargetState{Format: wgpu.TextureFormat(t.Format), WriteMask: wgpu.ColorWriteMask(t.WriteMask)}
			if t.Blend != nil {
				target.Blend = &wgpu.BlendState{
					Color: blendComponent(t.Blend.Color),
					Alpha: blendComponent(t.Blend.Alpha),
				}
			}
			targets[i] = target
		}
		desc.Fragment = &wgpu.FragmentState{Module: module.raw, EntryPoint: fs.EntryPoint, Targets: targets}
	}
	raw, err := d.raw.TryCreateRenderPipeline(desc)
	d.reportErr(err)
	return &renderPipeline{raw: raw}
}

func frontFace(f gpu.FrontFace) wgpu.FrontFace {
	if f == 0 {
		return wgpu.FrontFaceCCW
	}
	return wgpu.FrontFace(f)
}

func cullMode(c gpu.CullMode) wgpu.CullMode {
	if c == 0 {
		return wgpu.CullModeNone
	}
	return wgpu.CullMode(c)
}

func blendComponent(c gpu.BlendComponent) wgpu.BlendComponent {
	return wgpu.BlendComponent{
		Operation: wgpu.BlendOperation(c.Operation),
		SrcFactor: wgpu.BlendFactor(c.SrcFactor),
		DstFactor: wgpu.BlendFactor(c.DstFactor),
	}
}

func (d *device) CreateCommandEncoder(descriptor *gpu.CommandEncoderDescriptor) gpu.CommandEncoder {
	desc := &wgpu.CommandEncoderDescriptor{}
	if descriptor != nil {
		desc.Label = descriptor.Label
	}
	raw, err := d.raw.TryCreateCommandEncoder(desc)
	d.reportErr(err)
	return &commandEncoder{device: d, raw: raw}
}

func (d *device) Poll(wait bool) error {
	d.raw.Poll(wait, nil)
	return nil
}

func (d *device) Release() {
	d.raw.Release()
}

type queue struct {
	device *device
	raw    *wgpu.Queue
}

func (q *queue) WriteBuffer(b gpu.Buffer, offset uint64, data []byte) {
	buf, _ := b.(*buffer)
	if buf == nil || buf.raw == nil {
		q.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Buffer is invalid"})
		return
	}
	q.device.reportErr(q.raw.TryWriteBuffer(buf.raw, offset, data))
}

func copyStride(v uint32) uint32 {
	if v == 0 {
		return wgpu.CopyStrideUndefined
	}
	return v
}

func (q *queue) WriteTexture(t gpu.TexelCopyTextureInfo, data []byte, layout gpu.TexelCopyBufferLayout, size gpu.Extent3d) {
	tex, _ := t.Texture.(*texture)
	if tex == nil || tex.raw == nil {
		q.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Texture is invalid"})
		return
	}
	q.device.reportErr(q.raw.TryWriteTexture(
		texelCopyTexture(tex, t),
		data,
		&wgpu.TexelCopyBufferLayout{Offset: layout.Offset, BytesPerRow: copyStride(layout.BytesPerRow), RowsPerImage: copyStride(layout.RowsPerImage)},
		&wgpu.Extent3D{Width: size.Width, Height: size.Height, DepthOrArrayLayers: max(size.DepthOrArrayLayers, 1)},
	))
}

func texelCopyTexture(tex *texture, t gpu.TexelCopyTextureInfo) *wgpu.TexelCopyTextureInfo {
	aspect := wgpu.TextureAspect(t.Aspect)
	if aspect == 0 {
		aspect = wgpu.TextureAspectAll
	}
	return &wgpu.TexelCopyTextureInfo{
		Texture:  tex.raw,
		MipLevel: t.MipLevel,
		Origin:   wgpu.Origin3D{X: t.Origin.X, Y: t.Origin.Y, Z: t.Origin.Z},
		Aspect:   aspect,
	}
}

func (q *queue) Submit(commands ...gpu.CommandBuffer) gpu.SubmissionIndex {
	raw := make([]*wgpu.CommandBuffer, 0, len(commands))
	for _, c := range commands {
		cb, _ := c.(*commandBuffer)
		if cb == nil || cb.raw == nil {
			q.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "CommandBuffer is invalid"})
			return 0
		}
		raw = append(raw, cb.raw)
	}
	index := q.raw.Submit(raw...)
	for _, cb := range raw {
		cb.Release()
	}
	return gpu.SubmissionIndex(index)
}

func (q *queue) Release() {
	q.raw.Release()
}

type surface struct {
	raw    *wgpu.Surface
	device *device
}

func (s *surface) GetCapabilities(a gpu.Adapter) gpu.SurfaceCapabilities {
	ad, _ := a.(*adapter)
	if ad == nil {
		return gpu.SurfaceCapabilities{}
	}
	caps := s.raw.GetCapabilities(ad.raw)
	out := gpu.SurfaceCapabilities{Usages: gpu.TextureUsagesRenderAttachment}
	for _, f := range caps.Formats {
		out.Formats = append(out.Formats, gpu.TextureFormat(f))
	}
	for _, m := range caps.PresentModes {
		out.PresentModes = append(out.PresentModes, gpu.PresentMode(m))
	}
	for _, m := range caps.AlphaModes {
		out.AlphaModes = append(out.AlphaModes, gpu.CompositeAlphaMode(m))
	}
	return out
}

func (s *surface) Configure(dev gpu.Device, config *gpu.SurfaceConfiguration) {
	d, _ := dev.(*device)
	if d == nil {
		panic("wgpu: Surface.Configure needs a webgpu device")
	}
	if err := validateSurfaceConfiguration(config, s.GetCapabilities(d.adapter), d.raw.GetLimits().MaxTextureDimension2D); err != nil {
		d.reportErr(err)
		return
	}
	s.device = d
	viewFormats := make([]wgpu.TextureFormat, len(config.ViewFormats))
	for i, f := range config.ViewFormats {
		viewFormats[i] = wgpu.TextureFormat(f)
	}
	s.raw.Configure(d.raw, &wgpu.SurfaceConfiguration{
		Usage:                      wgpu.TextureUsage(config.Usage),
		Format:                     wgpu.TextureFormat(config.Format),
		Width:                      config.Width,
		Height:                     config.Height,
		PresentMode:                wgpu.PresentMode(config.PresentMode),
		AlphaMode:                  wgpu.CompositeAlphaMode(config.AlphaMode),
		ViewFormats:                viewFormats,
		DesiredMaximumFrameLatency: config.DesiredMaximumFrameLatency,
	})
}

func validateSurfaceConfiguration(config *gpu.SurfaceConfiguration, caps gpu.SurfaceCapabilities, maxTextureDimension2D uint32) error {
	for _, format := range config.ViewFormats {
		if format == config.Format {
			continue
		}
		if !slices.Contains(caps.Formats, config.Format) {
			return fmt.Errorf("Requested format %s is not in list of supported formats: %v", config.Format, caps.Formats)
		}
		if removeSrgbSuffix(config.Format) != removeSrgbSuffix(format) {
			return fmt.Errorf("The view format %s is not compatible with texture format %s, only changing srgb-ness is allowed.", format, config.Format)
		}
	}

	width, height := config.Width, config.Height
	if width > maxTextureDimension2D || height > maxTextureDimension2D {
		return fmt.Errorf("`Surface` width and height must be within the maximum supported texture size. Requested was (%d, %d), maximum extent for either dimension is %d.", width, height, maxTextureDimension2D)
	}

	presentMode := config.PresentMode
	if presentMode == 0 {
		presentMode = gpu.PresentModeFifo
	}
	if !slices.Contains(caps.PresentModes, presentMode) {
		return fmt.Errorf("Requested present mode %s is not in the list of supported present modes: %v", presentMode, caps.PresentModes)
	}
	if !slices.Contains(caps.Formats, config.Format) {
		return fmt.Errorf("Requested format %s is not in list of supported formats: %v", config.Format, caps.Formats)
	}
	if !slices.Contains(caps.AlphaModes, config.AlphaMode) {
		if config.AlphaMode != gpu.CompositeAlphaModeAuto ||
			!(slices.Contains(caps.AlphaModes, gpu.CompositeAlphaModeOpaque) || slices.Contains(caps.AlphaModes, gpu.CompositeAlphaModeInherit)) {
			return fmt.Errorf("Requested alpha mode %s is not in the list of supported alpha modes: %v", config.AlphaMode, caps.AlphaModes)
		}
	}
	if width == 0 || height == 0 {
		return errors.New("Both `Surface` width and height must be non-zero. Wait to recreate the `Surface` until the window has non-zero area.")
	}
	return nil
}

func removeSrgbSuffix(format gpu.TextureFormat) gpu.TextureFormat {
	if format.HasSrgbSuffix() {
		return format - 1
	}
	return format
}

func (s *surface) GetCurrentTexture() gpu.CurrentSurfaceTexture {
	st, err := s.raw.TryGetCurrentTexture()
	if err != nil {
		if s.device != nil {
			s.device.reportErr(err)
		}
		return gpu.CurrentSurfaceTexture{Status: gpu.CurrentSurfaceTextureValidation}
	}
	var status gpu.CurrentSurfaceTextureStatus
	switch st.Status {
	case wgpu.SurfaceGetCurrentTextureStatusSuccessOptimal:
		status = gpu.CurrentSurfaceTextureSuccess
	case wgpu.SurfaceGetCurrentTextureStatusSuccessSuboptimal:
		status = gpu.CurrentSurfaceTextureSuboptimal
	case wgpu.SurfaceGetCurrentTextureStatusTimeout:
		status = gpu.CurrentSurfaceTextureTimeout
	case wgpu.SurfaceGetCurrentTextureStatusOutdated:
		status = gpu.CurrentSurfaceTextureOutdated
	case wgpu.SurfaceGetCurrentTextureStatusLost:
		status = gpu.CurrentSurfaceTextureLost
	case wgpu.SurfaceGetCurrentTextureStatusOccluded:
		status = gpu.CurrentSurfaceTextureOccluded
	default:
		status = gpu.CurrentSurfaceTextureValidation
	}
	out := gpu.CurrentSurfaceTexture{Status: status}
	if st.Texture != nil {
		out.Texture = &texture{
			device: s.device,
			raw:    st.Texture,
			width:  st.Texture.GetWidth(),
			height: st.Texture.GetHeight(),
			format: gpu.TextureFormat(st.Texture.GetFormat()),
		}
	}
	return out
}

func (s *surface) Present() {
	if s.device == nil {
		return
	}
	s.raw.Present()
}

func (s *surface) Release() {
	s.raw.Release()
}

type texture struct {
	device *device
	raw    *wgpu.Texture
	width  uint32
	height uint32
	format gpu.TextureFormat
}

func (t *texture) CreateView(descriptor *gpu.TextureViewDescriptor) gpu.TextureView {
	if t.raw == nil {
		t.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Texture is invalid"})
		return &textureView{}
	}
	var desc *wgpu.TextureViewDescriptor
	if descriptor != nil {
		desc = &wgpu.TextureViewDescriptor{
			Label:           descriptor.Label,
			Format:          wgpu.TextureFormat(descriptor.Format),
			Dimension:       wgpu.TextureViewDimension(descriptor.Dimension),
			MipLevelCount:   wgpu.MipLevelCountUndefined,
			ArrayLayerCount: wgpu.ArrayLayerCountUndefined,
			Aspect:          wgpu.TextureAspectAll,
		}
	}
	raw, err := t.raw.TryCreateView(desc)
	t.device.reportErr(err)
	return &textureView{raw: raw}
}

func (t *texture) Width() uint32             { return t.width }
func (t *texture) Height() uint32            { return t.height }
func (t *texture) Format() gpu.TextureFormat { return t.format }
func (t *texture) Destroy() {
	if t.raw != nil {
		t.raw.Destroy()
	}
}
func (t *texture) Release() {
	if t.raw != nil {
		t.raw.Release()
	}
}

type textureView struct{ raw *wgpu.TextureView }

func (v *textureView) Release() {
	if v.raw != nil {
		v.raw.Release()
	}
}

type buffer struct {
	device *device
	raw    *wgpu.Buffer
	size   uint64
}

func (b *buffer) Size() uint64 { return b.size }

func (b *buffer) MapAsync(mode gpu.MapMode, offset, size uint64, callback func(err error)) {
	if b.raw == nil {
		b.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "Buffer is invalid"})
		return
	}
	err := b.raw.TryMapAsync(wgpu.MapMode(mode), offset, size, func(status wgpu.MapAsyncStatus) {
		if status == wgpu.MapAsyncStatusSuccess {
			callback(nil)
			return
		}
		callback(fmt.Errorf("buffer map failed: %s", status))
	})
	b.device.reportErr(err)
}

func (b *buffer) GetMappedRange(offset, size uint64) []byte {
	if b.raw == nil {
		return nil
	}
	return b.raw.GetMappedRange(uint(offset), uint(size))
}

func (b *buffer) Unmap() {
	if b.raw != nil {
		b.device.reportErr(b.raw.TryUnmap())
	}
}

func (b *buffer) Destroy() {
	if b.raw != nil {
		b.raw.Destroy()
	}
}

func (b *buffer) Release() {
	if b.raw != nil {
		b.raw.Release()
	}
}

type sampler struct{ raw *wgpu.Sampler }

func (s *sampler) Release() {
	if s.raw != nil {
		s.raw.Release()
	}
}

type shaderModule struct{ raw *wgpu.ShaderModule }

func (m *shaderModule) Release() {
	if m.raw != nil {
		m.raw.Release()
	}
}

type bindGroupLayout struct{ raw *wgpu.BindGroupLayout }

func (l *bindGroupLayout) Release() {
	if l.raw != nil {
		l.raw.Release()
	}
}

type bindGroup struct{ raw *wgpu.BindGroup }

func (g *bindGroup) Release() {
	if g.raw != nil {
		g.raw.Release()
	}
}

type pipelineLayout struct{ raw *wgpu.PipelineLayout }

func (l *pipelineLayout) Release() {
	if l.raw != nil {
		l.raw.Release()
	}
}

type renderPipeline struct{ raw *wgpu.RenderPipeline }

func (p *renderPipeline) Release() {
	if p.raw != nil {
		p.raw.Release()
	}
}

type commandBuffer struct{ raw *wgpu.CommandBuffer }

func (c *commandBuffer) Release() {
	if c.raw != nil {
		c.raw.Release()
	}
}

type commandEncoder struct {
	device *device
	raw    *wgpu.CommandEncoder
}

func (e *commandEncoder) BeginRenderPass(descriptor *gpu.RenderPassDescriptor) gpu.RenderPass {
	if e.raw == nil {
		e.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "CommandEncoder is invalid"})
		return &renderPass{device: e.device}
	}
	desc := &wgpu.RenderPassDescriptor{Label: descriptor.Label}
	for _, a := range descriptor.ColorAttachments {
		view, _ := a.View.(*textureView)
		if view == nil || view.raw == nil {
			e.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "TextureView is invalid"})
			return &renderPass{device: e.device}
		}
		attachment := wgpu.RenderPassColorAttachment{
			View:       view.raw,
			LoadOp:     wgpu.LoadOp(a.LoadOp),
			StoreOp:    wgpu.StoreOp(a.StoreOp),
			ClearValue: wgpu.Color{R: a.ClearValue.R, G: a.ClearValue.G, B: a.ClearValue.B, A: a.ClearValue.A},
		}
		if a.ResolveTarget != nil {
			if resolve, _ := a.ResolveTarget.(*textureView); resolve != nil {
				attachment.ResolveTarget = resolve.raw
			}
		}
		desc.ColorAttachments = append(desc.ColorAttachments, attachment)
	}
	if ds := descriptor.DepthStencilAttachment; ds != nil {
		view, _ := ds.View.(*textureView)
		if view == nil || view.raw == nil {
			e.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "TextureView is invalid"})
			return &renderPass{device: e.device}
		}
		desc.DepthStencilAttachment = &wgpu.RenderPassDepthStencilAttachment{
			View:            view.raw,
			DepthLoadOp:     wgpu.LoadOp(ds.DepthLoadOp),
			DepthStoreOp:    wgpu.StoreOp(ds.DepthStoreOp),
			DepthClearValue: ds.DepthClearValue,
		}
	}
	raw, err := e.raw.TryBeginRenderPass(desc)
	e.device.reportErr(err)
	return &renderPass{device: e.device, raw: raw}
}

func (e *commandEncoder) CopyTextureToBuffer(source gpu.TexelCopyTextureInfo, destination gpu.TexelCopyBufferInfo, size gpu.Extent3d) {
	tex, _ := source.Texture.(*texture)
	buf, _ := destination.Buffer.(*buffer)
	if e.raw == nil || tex == nil || tex.raw == nil || buf == nil || buf.raw == nil {
		e.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "CopyTextureToBuffer resource is invalid"})
		return
	}
	e.device.reportErr(e.raw.TryCopyTextureToBuffer(
		texelCopyTexture(tex, source),
		&wgpu.TexelCopyBufferInfo{
			Buffer: buf.raw,
			Layout: wgpu.TexelCopyBufferLayout{
				Offset:       destination.Layout.Offset,
				BytesPerRow:  copyStride(destination.Layout.BytesPerRow),
				RowsPerImage: copyStride(destination.Layout.RowsPerImage),
			},
		},
		&wgpu.Extent3D{Width: size.Width, Height: size.Height, DepthOrArrayLayers: max(size.DepthOrArrayLayers, 1)},
	))
}

func (e *commandEncoder) Finish() gpu.CommandBuffer {
	if e.raw == nil {
		e.device.report(&gpu.Error{Filter: gpu.ErrorFilterValidation, Message: "CommandEncoder is invalid"})
		return &commandBuffer{}
	}
	raw, err := e.raw.TryFinish(nil)
	e.device.reportErr(err)
	return &commandBuffer{raw: raw}
}

func (e *commandEncoder) Release() {
	if e.raw != nil {
		e.raw.Release()
	}
}

type renderPass struct {
	device *device
	raw    *wgpu.RenderPassEncoder
}

func (p *renderPass) SetPipeline(pipeline gpu.RenderPipeline) {
	rp, _ := pipeline.(*renderPipeline)
	if p.raw == nil || rp == nil || rp.raw == nil {
		return
	}
	p.raw.SetPipeline(rp.raw)
}

func (p *renderPass) SetBindGroup(index uint32, group gpu.BindGroup, offsets []uint32) {
	bg, _ := group.(*bindGroup)
	if p.raw == nil || bg == nil || bg.raw == nil {
		return
	}
	p.raw.SetBindGroup(index, bg.raw, offsets)
}

func (p *renderPass) SetViewport(x, y, width, height, minDepth, maxDepth float32) {
	if p.raw != nil {
		p.raw.SetViewport(x, y, width, height, minDepth, maxDepth)
	}
}

func (p *renderPass) SetScissorRect(x, y, width, height uint32) {
	if p.raw != nil {
		p.raw.SetScissorRect(x, y, width, height)
	}
}

func (p *renderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	if p.raw != nil {
		p.raw.Draw(vertexCount, instanceCount, firstVertex, firstInstance)
	}
}

func (p *renderPass) End() {
	if p.raw == nil {
		return
	}
	p.device.reportErr(p.raw.TryEnd())
	p.raw.Release()
}
