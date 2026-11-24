//go:build linux || js

// Package gpu is the subset of the WebGPU API the wgpu renderer and atlas use. It mirrors
// wgpu's Rust API; internal/gpu/webgpu implements it over wgpu-native and browser WebGPU,
// internal/gpu/webgl2 over WebGL2.
package gpu

import "fmt"

// Backend is the graphics API an adapter runs on (wgpu::Backend).
type Backend uint8

const (
	BackendNoop Backend = iota
	BackendVulkan
	BackendMetal
	BackendDx12
	BackendGl
	BackendBrowserWebGpu
)

// String returns wgpu's Debug name of the backend.
func (b Backend) String() string {
	switch b {
	case BackendNoop:
		return "Noop"
	case BackendVulkan:
		return "Vulkan"
	case BackendMetal:
		return "Metal"
	case BackendDx12:
		return "Dx12"
	case BackendGl:
		return "Gl"
	case BackendBrowserWebGpu:
		return "BrowserWebGpu"
	}
	return fmt.Sprintf("Backend(%d)", uint8(b))
}

// Backends is a set of backends (wgpu::Backends).
type Backends uint32

const (
	BackendsVulkan        Backends = 1 << 0
	BackendsGl            Backends = 1 << 1
	BackendsMetal         Backends = 1 << 2
	BackendsDx12          Backends = 1 << 3
	BackendsBrowserWebGpu Backends = 1 << 5
	BackendsAll                    = BackendsVulkan | BackendsGl | BackendsMetal | BackendsDx12 | BackendsBrowserWebGpu
)

// DeviceType is the kind of adapter (wgpu::DeviceType).
type DeviceType uint8

const (
	DeviceTypeOther DeviceType = iota
	DeviceTypeIntegratedGpu
	DeviceTypeDiscreteGpu
	DeviceTypeVirtualGpu
	DeviceTypeCpu
)

// String returns wgpu's Debug name of the device type.
func (t DeviceType) String() string {
	switch t {
	case DeviceTypeOther:
		return "Other"
	case DeviceTypeIntegratedGpu:
		return "IntegratedGpu"
	case DeviceTypeDiscreteGpu:
		return "DiscreteGpu"
	case DeviceTypeVirtualGpu:
		return "VirtualGpu"
	case DeviceTypeCpu:
		return "Cpu"
	}
	return fmt.Sprintf("DeviceType(%d)", uint8(t))
}

// AdapterInfo describes an adapter (wgpu::AdapterInfo).
type AdapterInfo struct {
	Name       string
	Vendor     uint32
	Device     uint32
	DeviceType DeviceType
	Driver     string
	DriverInfo string
	Backend    Backend
}

// Features is a set of optional device features (wgpu::Features).
type Features uint64

const (
	FeaturesDualSourceBlending Features = 1 << 0
)

// Contains reports whether every feature of other is in f.
func (f Features) Contains(other Features) bool {
	return f&other == other
}

// DownlevelFlags are capabilities below the WebGPU baseline (wgpu::DownlevelFlags).
type DownlevelFlags uint64

const (
	DownlevelFlagsVertexStorage DownlevelFlags = 1 << iota
	DownlevelFlagsFragmentStorage
)

// Contains reports whether every flag of other is in f.
func (f DownlevelFlags) Contains(other DownlevelFlags) bool {
	return f&other == other
}

// DownlevelCapabilities are an adapter's downlevel capabilities (wgpu::DownlevelCapabilities).
type DownlevelCapabilities struct {
	Flags DownlevelFlags
}

// Limits are device limits (wgpu::Limits, the fields WebGPU and wgpu-native share).
type Limits struct {
	MaxTextureDimension1D                     uint32
	MaxTextureDimension2D                     uint32
	MaxTextureDimension3D                     uint32
	MaxTextureArrayLayers                     uint32
	MaxBindGroups                             uint32
	MaxBindingsPerBindGroup                   uint32
	MaxDynamicUniformBuffersPerPipelineLayout uint32
	MaxDynamicStorageBuffersPerPipelineLayout uint32
	MaxSampledTexturesPerShaderStage          uint32
	MaxSamplersPerShaderStage                 uint32
	MaxStorageBuffersPerShaderStage           uint32
	MaxStorageTexturesPerShaderStage          uint32
	MaxUniformBuffersPerShaderStage           uint32
	MaxUniformBufferBindingSize               uint64
	MaxStorageBufferBindingSize               uint64
	MinUniformBufferOffsetAlignment           uint32
	MinStorageBufferOffsetAlignment           uint32
	MaxVertexBuffers                          uint32
	MaxBufferSize                             uint64
	MaxVertexAttributes                       uint32
	MaxVertexBufferArrayStride                uint32
	MaxInterStageShaderVariables              uint32
	MaxColorAttachments                       uint32
	MaxColorAttachmentBytesPerSample          uint32
	MaxComputeWorkgroupStorageSize            uint32
	MaxComputeInvocationsPerWorkgroup         uint32
	MaxComputeWorkgroupSizeX                  uint32
	MaxComputeWorkgroupSizeY                  uint32
	MaxComputeWorkgroupSizeZ                  uint32
	MaxComputeWorkgroupsPerDimension          uint32
	MaxImmediateSize                          uint32
	MaxNonSamplerBindings                     uint32
}

// DefaultLimits returns wgpu's Limits::defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxTextureDimension1D:                     8192,
		MaxTextureDimension2D:                     8192,
		MaxTextureDimension3D:                     2048,
		MaxTextureArrayLayers:                     256,
		MaxBindGroups:                             4,
		MaxBindingsPerBindGroup:                   1000,
		MaxDynamicUniformBuffersPerPipelineLayout: 8,
		MaxDynamicStorageBuffersPerPipelineLayout: 4,
		MaxSampledTexturesPerShaderStage:          16,
		MaxSamplersPerShaderStage:                 16,
		MaxStorageBuffersPerShaderStage:           8,
		MaxStorageTexturesPerShaderStage:          4,
		MaxUniformBuffersPerShaderStage:           12,
		MaxUniformBufferBindingSize:               64 << 10,
		MaxStorageBufferBindingSize:               128 << 20,
		MaxVertexBuffers:                          8,
		MaxBufferSize:                             256 << 20,
		MaxVertexAttributes:                       16,
		MaxVertexBufferArrayStride:                2048,
		MaxInterStageShaderVariables:              16,
		MinUniformBufferOffsetAlignment:           256,
		MinStorageBufferOffsetAlignment:           256,
		MaxColorAttachments:                       8,
		MaxColorAttachmentBytesPerSample:          32,
		MaxComputeWorkgroupStorageSize:            16384,
		MaxComputeInvocationsPerWorkgroup:         256,
		MaxComputeWorkgroupSizeX:                  256,
		MaxComputeWorkgroupSizeY:                  256,
		MaxComputeWorkgroupSizeZ:                  64,
		MaxComputeWorkgroupsPerDimension:          65535,
		MaxImmediateSize:                          0,
		MaxNonSamplerBindings:                     1_000_000,
	}
}

// DownlevelDefaults returns wgpu's Limits::downlevel_defaults.
func DownlevelDefaults() Limits {
	l := DefaultLimits()
	l.MaxTextureDimension1D = 2048
	l.MaxTextureDimension2D = 2048
	l.MaxTextureDimension3D = 256
	l.MaxStorageBuffersPerShaderStage = 4
	l.MaxUniformBufferBindingSize = 16 << 10
	l.MaxInterStageShaderVariables = 15
	l.MaxColorAttachments = 4
	l.MaxComputeWorkgroupStorageSize = 16352
	return l
}

// DownlevelWebGL2Defaults returns wgpu's Limits::downlevel_webgl2_defaults.
func DownlevelWebGL2Defaults() Limits {
	l := DownlevelDefaults()
	l.MaxUniformBuffersPerShaderStage = 11
	l.MaxStorageBuffersPerShaderStage = 0
	l.MaxStorageTexturesPerShaderStage = 0
	l.MaxDynamicStorageBuffersPerPipelineLayout = 0
	l.MaxStorageBufferBindingSize = 0
	l.MaxVertexBufferArrayStride = 255
	l.MaxComputeWorkgroupStorageSize = 0
	l.MaxComputeInvocationsPerWorkgroup = 0
	l.MaxComputeWorkgroupSizeX = 0
	l.MaxComputeWorkgroupSizeY = 0
	l.MaxComputeWorkgroupSizeZ = 0
	l.MaxComputeWorkgroupsPerDimension = 0
	l.MaxInterStageShaderVariables = 15
	return l
}

// UsingResolution takes the texture dimension limits from other.
func (l Limits) UsingResolution(other Limits) Limits {
	l.MaxTextureDimension1D = other.MaxTextureDimension1D
	l.MaxTextureDimension2D = other.MaxTextureDimension2D
	l.MaxTextureDimension3D = other.MaxTextureDimension3D
	return l
}

// UsingAlignment takes the buffer offset alignment limits from other.
func (l Limits) UsingAlignment(other Limits) Limits {
	l.MinUniformBufferOffsetAlignment = other.MinUniformBufferOffsetAlignment
	l.MinStorageBufferOffsetAlignment = other.MinStorageBufferOffsetAlignment
	return l
}

// TextureFormat is a texture format; values are webgpu.h's WGPUTextureFormat.
type TextureFormat uint32

const (
	TextureFormatUndefined      TextureFormat = 0x00
	TextureFormatR8Unorm        TextureFormat = 0x01
	TextureFormatRgba8Unorm     TextureFormat = 0x16
	TextureFormatRgba8UnormSrgb TextureFormat = 0x17
	TextureFormatBgra8Unorm     TextureFormat = 0x1B
	TextureFormatBgra8UnormSrgb TextureFormat = 0x1C
	TextureFormatRgba32Uint     TextureFormat = 0x2A
	TextureFormatDepth32Float   TextureFormat = 0x30
)

var srgbFormats = map[TextureFormat]bool{
	0x17: true, 0x1C: true, 0x33: true, 0x35: true, 0x37: true, 0x3F: true, 0x41: true, 0x43: true, 0x45: true,
	0x4B: true, 0x4D: true, 0x4F: true, 0x51: true, 0x53: true, 0x55: true, 0x57: true, 0x59: true, 0x5B: true,
	0x5D: true, 0x5F: true, 0x61: true, 0x63: true, 0x65: true,
}

// HasSrgbSuffix reports whether the format is an sRGB format.
func (f TextureFormat) HasSrgbSuffix() bool {
	return srgbFormats[f]
}

// String returns wgpu's Debug name for the formats gpui uses.
func (f TextureFormat) String() string {
	switch f {
	case TextureFormatR8Unorm:
		return "R8Unorm"
	case TextureFormatRgba8Unorm:
		return "Rgba8Unorm"
	case TextureFormatRgba8UnormSrgb:
		return "Rgba8UnormSrgb"
	case TextureFormatBgra8Unorm:
		return "Bgra8Unorm"
	case TextureFormatBgra8UnormSrgb:
		return "Bgra8UnormSrgb"
	case TextureFormatRgba32Uint:
		return "Rgba32Uint"
	case TextureFormatDepth32Float:
		return "Depth32Float"
	}
	return fmt.Sprintf("TextureFormat(%#x)", uint32(f))
}

// TextureUsages is a set of texture usages; values are webgpu.h's WGPUTextureUsage.
type TextureUsages uint64

const (
	TextureUsagesCopySrc             TextureUsages = 0x01
	TextureUsagesCopyDst             TextureUsages = 0x02
	TextureUsagesTextureBinding      TextureUsages = 0x04
	TextureUsagesStorageBinding      TextureUsages = 0x08
	TextureUsagesRenderAttachment    TextureUsages = 0x10
	TextureUsagesTransientAttachment TextureUsages = 0x20
)

// Contains reports whether every usage of other is in u.
func (u TextureUsages) Contains(other TextureUsages) bool {
	return u&other == other
}

// String formats the usages as wgpu's bitflags Debug does.
func (u TextureUsages) String() string {
	names := []struct {
		bit  TextureUsages
		name string
	}{
		{TextureUsagesCopySrc, "COPY_SRC"},
		{TextureUsagesCopyDst, "COPY_DST"},
		{TextureUsagesTextureBinding, "TEXTURE_BINDING"},
		{TextureUsagesStorageBinding, "STORAGE_BINDING"},
		{TextureUsagesRenderAttachment, "RENDER_ATTACHMENT"},
		{TextureUsagesTransientAttachment, "TRANSIENT_ATTACHMENT"},
	}
	s := ""
	for _, n := range names {
		if u&n.bit != 0 {
			if s != "" {
				s += " | "
			}
			s += n.name
		}
	}
	return "TextureUsages(" + s + ")"
}

// TextureFormatFeatures are the usages an adapter allows for a format (wgpu::TextureFormatFeatures).
type TextureFormatFeatures struct {
	AllowedUsages TextureUsages
}

// BufferUsages is a set of buffer usages; values are webgpu.h's WGPUBufferUsage.
type BufferUsages uint64

const (
	BufferUsagesMapRead  BufferUsages = 0x0001
	BufferUsagesMapWrite BufferUsages = 0x0002
	BufferUsagesCopySrc  BufferUsages = 0x0004
	BufferUsagesCopyDst  BufferUsages = 0x0008
	BufferUsagesIndex    BufferUsages = 0x0010
	BufferUsagesVertex   BufferUsages = 0x0020
	BufferUsagesUniform  BufferUsages = 0x0040
	BufferUsagesStorage  BufferUsages = 0x0080
)

// ShaderStages is a set of shader stages; values are webgpu.h's WGPUShaderStage.
type ShaderStages uint64

const (
	ShaderStagesVertex         ShaderStages = 0x1
	ShaderStagesFragment       ShaderStages = 0x2
	ShaderStagesVertexFragment              = ShaderStagesVertex | ShaderStagesFragment
)

// ColorWrites is a color write mask; values are webgpu.h's WGPUColorWriteMask.
type ColorWrites uint64

const (
	ColorWritesRed   ColorWrites = 0x1
	ColorWritesGreen ColorWrites = 0x2
	ColorWritesBlue  ColorWrites = 0x4
	ColorWritesAlpha ColorWrites = 0x8
	ColorWritesColor             = ColorWritesRed | ColorWritesGreen | ColorWritesBlue
	ColorWritesAll               = ColorWritesColor | ColorWritesAlpha
)

// MapMode selects buffer mapping access; values are webgpu.h's WGPUMapMode.
type MapMode uint64

const (
	MapModeRead  MapMode = 0x1
	MapModeWrite MapMode = 0x2
)

// CompositeAlphaMode is a surface alpha mode; values are webgpu.h's WGPUCompositeAlphaMode.
type CompositeAlphaMode uint32

const (
	CompositeAlphaModeAuto           CompositeAlphaMode = 0
	CompositeAlphaModeOpaque         CompositeAlphaMode = 1
	CompositeAlphaModePreMultiplied  CompositeAlphaMode = 2
	CompositeAlphaModePostMultiplied CompositeAlphaMode = 3
	CompositeAlphaModeInherit        CompositeAlphaMode = 4
)

// String returns wgpu's Debug name for the alpha mode.
func (m CompositeAlphaMode) String() string {
	switch m {
	case CompositeAlphaModeAuto:
		return "Auto"
	case CompositeAlphaModeOpaque:
		return "Opaque"
	case CompositeAlphaModePreMultiplied:
		return "PreMultiplied"
	case CompositeAlphaModePostMultiplied:
		return "PostMultiplied"
	case CompositeAlphaModeInherit:
		return "Inherit"
	}
	return fmt.Sprintf("CompositeAlphaMode(%#x)", uint32(m))
}

// PresentMode is a surface presentation mode; values are webgpu.h's WGPUPresentMode.
type PresentMode uint32

const (
	PresentModeFifo        PresentMode = 1
	PresentModeFifoRelaxed PresentMode = 2
	PresentModeImmediate   PresentMode = 3
	PresentModeMailbox     PresentMode = 4
)

// String returns wgpu's Debug name for the present mode.
func (m PresentMode) String() string {
	switch m {
	case PresentModeFifo:
		return "Fifo"
	case PresentModeFifoRelaxed:
		return "FifoRelaxed"
	case PresentModeImmediate:
		return "Immediate"
	case PresentModeMailbox:
		return "Mailbox"
	}
	return fmt.Sprintf("PresentMode(%#x)", uint32(m))
}

// TextureDimension is a texture's dimensionality; values are webgpu.h's.
type TextureDimension uint32

const (
	TextureDimensionD1 TextureDimension = 1
	TextureDimensionD2 TextureDimension = 2
	TextureDimensionD3 TextureDimension = 3
)

// TextureViewDimension is a view's dimensionality; values are webgpu.h's.
type TextureViewDimension uint32

const (
	TextureViewDimensionD2 TextureViewDimension = 2
)

// TextureAspect selects texture aspects; values are webgpu.h's.
type TextureAspect uint32

const (
	TextureAspectAll TextureAspect = 1
)

// TextureSampleType is a texture binding's sample type; values are webgpu.h's.
type TextureSampleType uint32

const (
	TextureSampleTypeFloat             TextureSampleType = 2
	TextureSampleTypeUnfilterableFloat TextureSampleType = 3
	TextureSampleTypeDepth             TextureSampleType = 4
	TextureSampleTypeSint              TextureSampleType = 5
	TextureSampleTypeUint              TextureSampleType = 6
)

// BufferBindingType is a buffer binding's type; values are webgpu.h's.
type BufferBindingType uint32

const (
	BufferBindingTypeUniform         BufferBindingType = 2
	BufferBindingTypeStorage         BufferBindingType = 3
	BufferBindingTypeReadOnlyStorage BufferBindingType = 4
)

// SamplerBindingType is a sampler binding's type; values are webgpu.h's.
type SamplerBindingType uint32

const (
	SamplerBindingTypeFiltering    SamplerBindingType = 2
	SamplerBindingTypeNonFiltering SamplerBindingType = 3
	SamplerBindingTypeComparison   SamplerBindingType = 4
)

// FilterMode is a sampler filter; values are webgpu.h's.
type FilterMode uint32

const (
	FilterModeNearest FilterMode = 1
	FilterModeLinear  FilterMode = 2
)

// AddressMode is a sampler address mode; values are webgpu.h's.
type AddressMode uint32

const (
	AddressModeClampToEdge  AddressMode = 1
	AddressModeRepeat       AddressMode = 2
	AddressModeMirrorRepeat AddressMode = 3
)

// PrimitiveTopology is a primitive topology; values are webgpu.h's.
type PrimitiveTopology uint32

const (
	PrimitiveTopologyPointList     PrimitiveTopology = 1
	PrimitiveTopologyLineList      PrimitiveTopology = 2
	PrimitiveTopologyLineStrip     PrimitiveTopology = 3
	PrimitiveTopologyTriangleList  PrimitiveTopology = 4
	PrimitiveTopologyTriangleStrip PrimitiveTopology = 5
)

// FrontFace is a winding order; values are webgpu.h's.
type FrontFace uint32

const (
	FrontFaceCcw FrontFace = 1
	FrontFaceCw  FrontFace = 2
)

// CullMode is a face culling mode; values are webgpu.h's.
type CullMode uint32

const (
	CullModeNone  CullMode = 1
	CullModeFront CullMode = 2
	CullModeBack  CullMode = 3
)

// BlendFactor is a blend factor; values are webgpu.h's.
type BlendFactor uint32

const (
	BlendFactorZero              BlendFactor = 0x01
	BlendFactorOne               BlendFactor = 0x02
	BlendFactorSrc               BlendFactor = 0x03
	BlendFactorOneMinusSrc       BlendFactor = 0x04
	BlendFactorSrcAlpha          BlendFactor = 0x05
	BlendFactorOneMinusSrcAlpha  BlendFactor = 0x06
	BlendFactorDst               BlendFactor = 0x07
	BlendFactorOneMinusDst       BlendFactor = 0x08
	BlendFactorDstAlpha          BlendFactor = 0x09
	BlendFactorOneMinusDstAlpha  BlendFactor = 0x0A
	BlendFactorSrc1              BlendFactor = 0x0E
	BlendFactorOneMinusSrc1      BlendFactor = 0x0F
	BlendFactorSrc1Alpha         BlendFactor = 0x10
	BlendFactorOneMinusSrc1Alpha BlendFactor = 0x11
)

// BlendOperation is a blend operation; values are webgpu.h's.
type BlendOperation uint32

const (
	BlendOperationAdd             BlendOperation = 1
	BlendOperationSubtract        BlendOperation = 2
	BlendOperationReverseSubtract BlendOperation = 3
	BlendOperationMin             BlendOperation = 4
	BlendOperationMax             BlendOperation = 5
)

// CompareFunction is a depth or sampler comparison; values are webgpu.h's.
type CompareFunction uint32

const (
	CompareFunctionUndefined    CompareFunction = 0
	CompareFunctionNever        CompareFunction = 1
	CompareFunctionLess         CompareFunction = 2
	CompareFunctionEqual        CompareFunction = 3
	CompareFunctionLessEqual    CompareFunction = 4
	CompareFunctionGreater      CompareFunction = 5
	CompareFunctionNotEqual     CompareFunction = 6
	CompareFunctionGreaterEqual CompareFunction = 7
	CompareFunctionAlways       CompareFunction = 8
)

// LoadOp is a render pass load operation; values are webgpu.h's.
type LoadOp uint32

const (
	LoadOpLoad  LoadOp = 1
	LoadOpClear LoadOp = 2
)

// StoreOp is a render pass store operation; values are webgpu.h's.
type StoreOp uint32

const (
	StoreOpStore   StoreOp = 1
	StoreOpDiscard StoreOp = 2
)

// DeviceLostReason is why a device was lost (wgpu::DeviceLostReason).
type DeviceLostReason uint8

const (
	DeviceLostReasonUnknown DeviceLostReason = iota
	DeviceLostReasonDestroyed
)

// String returns wgpu's Debug name of the reason.
func (r DeviceLostReason) String() string {
	if r == DeviceLostReasonDestroyed {
		return "Destroyed"
	}
	return "Unknown"
}

// ErrorFilter selects the errors an error scope captures (wgpu::ErrorFilter).
type ErrorFilter uint8

const (
	ErrorFilterValidation ErrorFilter = iota
	ErrorFilterOutOfMemory
	ErrorFilterInternal
)

// Error is a device error delivered to an error scope or the uncaptured-error handler (wgpu::Error).
type Error struct {
	Filter  ErrorFilter
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

// PowerPreference is the adapter power preference (wgpu::PowerPreference).
type PowerPreference uint8

const (
	PowerPreferenceNone PowerPreference = iota
	PowerPreferenceLowPower
	PowerPreferenceHighPerformance
)

// Color is an RGBA clear color.
type Color struct {
	R, G, B, A float64
}

var (
	ColorTransparent = Color{}
	ColorBlack       = Color{A: 1}
)

// Extent3d is a texture extent.
type Extent3d struct {
	Width              uint32
	Height             uint32
	DepthOrArrayLayers uint32
}

// Origin3d is a texel origin.
type Origin3d struct {
	X, Y, Z uint32
}

// CopyBytesPerRowAlignment is wgpu::COPY_BYTES_PER_ROW_ALIGNMENT.
const CopyBytesPerRowAlignment = 256

// SurfaceTarget names the native window a surface is created for; only the fields of one
// windowing system are set.
type SurfaceTarget struct {
	WaylandDisplay uintptr
	WaylandSurface uintptr
	XcbConnection  uintptr
	XcbWindow      uint32
	XlibDisplay    uintptr
	XlibWindow     uint32
	MetalLayer     uintptr
	Win32HWND      uintptr
	Win32HInstance uintptr
	CanvasID       string
}

// InstanceDescriptor configures an instance.
type InstanceDescriptor struct {
	Backends Backends
}

// RequestAdapterOptions configures adapter selection.
type RequestAdapterOptions struct {
	PowerPreference      PowerPreference
	CompatibleSurface    Surface
	ForceFallbackAdapter bool
}

// DeviceDescriptor configures a device request.
type DeviceDescriptor struct {
	Label            string
	RequiredFeatures Features
	RequiredLimits   Limits
}

// TextureDescriptor describes a texture.
type TextureDescriptor struct {
	Label         string
	Size          Extent3d
	MipLevelCount uint32
	SampleCount   uint32
	Dimension     TextureDimension
	Format        TextureFormat
	Usage         TextureUsages
	ViewFormats   []TextureFormat
}

// TextureViewDescriptor describes a texture view; a nil descriptor is wgpu's default.
type TextureViewDescriptor struct {
	Label     string
	Format    TextureFormat
	Dimension TextureViewDimension
}

// BufferDescriptor describes a buffer.
type BufferDescriptor struct {
	Label            string
	Size             uint64
	Usage            BufferUsages
	MappedAtCreation bool
}

// SamplerDescriptor describes a sampler.
type SamplerDescriptor struct {
	Label        string
	AddressModeU AddressMode
	AddressModeV AddressMode
	AddressModeW AddressMode
	MagFilter    FilterMode
	MinFilter    FilterMode
	MipmapFilter FilterMode
	LodMinClamp  float32
	LodMaxClamp  float32
	Compare      CompareFunction
}

// ShaderModuleDescriptor describes a WGSL shader module.
type ShaderModuleDescriptor struct {
	Label string
	WGSL  string
}

// BindingType is a bind group layout entry's binding; exactly one field is set.
type BindingType struct {
	Buffer  *BufferBindingLayout
	Sampler *SamplerBindingLayout
	Texture *TextureBindingLayout
}

// BufferBindingLayout is wgpu::BindingType::Buffer.
type BufferBindingLayout struct {
	Type             BufferBindingType
	HasDynamicOffset bool
	MinBindingSize   uint64
}

// SamplerBindingLayout is wgpu::BindingType::Sampler.
type SamplerBindingLayout struct {
	Type SamplerBindingType
}

// TextureBindingLayout is wgpu::BindingType::Texture.
type TextureBindingLayout struct {
	SampleType    TextureSampleType
	ViewDimension TextureViewDimension
	Multisampled  bool
}

// BindGroupLayoutEntry is one binding of a bind group layout.
type BindGroupLayoutEntry struct {
	Binding    uint32
	Visibility ShaderStages
	Type       BindingType
}

// BindGroupLayoutDescriptor describes a bind group layout.
type BindGroupLayoutDescriptor struct {
	Label   string
	Entries []BindGroupLayoutEntry
}

// BindingResource is a bind group entry's resource; exactly one of Buffer, Sampler or
// TextureView is set. A zero Size binds the rest of the buffer.
type BindingResource struct {
	Buffer      Buffer
	Offset      uint64
	Size        uint64
	Sampler     Sampler
	TextureView TextureView
}

// BindGroupEntry is one binding of a bind group.
type BindGroupEntry struct {
	Binding  uint32
	Resource BindingResource
}

// BindGroupDescriptor describes a bind group.
type BindGroupDescriptor struct {
	Label   string
	Layout  BindGroupLayout
	Entries []BindGroupEntry
}

// PipelineLayoutDescriptor describes a pipeline layout.
type PipelineLayoutDescriptor struct {
	Label            string
	BindGroupLayouts []BindGroupLayout
}

// BlendComponent is one channel group's blend equation.
type BlendComponent struct {
	SrcFactor BlendFactor
	DstFactor BlendFactor
	Operation BlendOperation
}

// BlendState is a color target's blend state.
type BlendState struct {
	Color BlendComponent
	Alpha BlendComponent
}

var (
	BlendStateAlphaBlending = BlendState{
		Color: BlendComponent{SrcFactor: BlendFactorSrcAlpha, DstFactor: BlendFactorOneMinusSrcAlpha, Operation: BlendOperationAdd},
		Alpha: BlendComponent{SrcFactor: BlendFactorOne, DstFactor: BlendFactorOneMinusSrcAlpha, Operation: BlendOperationAdd},
	}
	BlendStatePremultipliedAlphaBlending = BlendState{
		Color: BlendComponent{SrcFactor: BlendFactorOne, DstFactor: BlendFactorOneMinusSrcAlpha, Operation: BlendOperationAdd},
		Alpha: BlendComponent{SrcFactor: BlendFactorOne, DstFactor: BlendFactorOneMinusSrcAlpha, Operation: BlendOperationAdd},
	}
)

// ColorTargetState is a render pipeline color target.
type ColorTargetState struct {
	Format    TextureFormat
	Blend     *BlendState
	WriteMask ColorWrites
}

// VertexState is a pipeline's vertex stage; gpui pulls instance data from bindings, so it
// has no vertex buffers.
type VertexState struct {
	Module     ShaderModule
	EntryPoint string
}

// FragmentState is a pipeline's fragment stage.
type FragmentState struct {
	Module     ShaderModule
	EntryPoint string
	Targets    []ColorTargetState
}

// PrimitiveState is a pipeline's primitive assembly state.
type PrimitiveState struct {
	Topology  PrimitiveTopology
	FrontFace FrontFace
	CullMode  CullMode
}

// DepthStencilState is a pipeline's depth state.
type DepthStencilState struct {
	Format              TextureFormat
	DepthWriteEnabled   bool
	DepthCompare        CompareFunction
	DepthBias           int32
	DepthBiasSlopeScale float32
	DepthBiasClamp      float32
}

// MultisampleState is a pipeline's multisample state; zero Count means 1.
type MultisampleState struct {
	Count                  uint32
	Mask                   uint32
	AlphaToCoverageEnabled bool
}

// RenderPipelineDescriptor describes a render pipeline.
type RenderPipelineDescriptor struct {
	Label        string
	Layout       PipelineLayout
	Vertex       VertexState
	Primitive    PrimitiveState
	DepthStencil *DepthStencilState
	Multisample  MultisampleState
	Fragment     *FragmentState
}

// CommandEncoderDescriptor describes a command encoder.
type CommandEncoderDescriptor struct {
	Label string
}

// RenderPassColorAttachment is a render pass color attachment.
type RenderPassColorAttachment struct {
	View          TextureView
	ResolveTarget TextureView
	LoadOp        LoadOp
	StoreOp       StoreOp
	ClearValue    Color
}

// RenderPassDepthStencilAttachment is a render pass depth attachment.
type RenderPassDepthStencilAttachment struct {
	View            TextureView
	DepthLoadOp     LoadOp
	DepthStoreOp    StoreOp
	DepthClearValue float32
}

// RenderPassDescriptor describes a render pass.
type RenderPassDescriptor struct {
	Label                  string
	ColorAttachments       []RenderPassColorAttachment
	DepthStencilAttachment *RenderPassDepthStencilAttachment
}

// TexelCopyTextureInfo names a texture region for copies.
type TexelCopyTextureInfo struct {
	Texture  Texture
	MipLevel uint32
	Origin   Origin3d
	Aspect   TextureAspect
}

// TexelCopyBufferLayout is a buffer's texel layout; zero BytesPerRow or RowsPerImage means None.
type TexelCopyBufferLayout struct {
	Offset       uint64
	BytesPerRow  uint32
	RowsPerImage uint32
}

// TexelCopyBufferInfo names a buffer region for copies.
type TexelCopyBufferInfo struct {
	Buffer Buffer
	Layout TexelCopyBufferLayout
}

// SurfaceCapabilities are what a surface supports on an adapter.
type SurfaceCapabilities struct {
	Formats      []TextureFormat
	PresentModes []PresentMode
	AlphaModes   []CompositeAlphaMode
	Usages       TextureUsages
}

// SurfaceConfiguration configures a surface.
type SurfaceConfiguration struct {
	Usage                      TextureUsages
	Format                     TextureFormat
	Width                      uint32
	Height                     uint32
	PresentMode                PresentMode
	DesiredMaximumFrameLatency uint32
	AlphaMode                  CompositeAlphaMode
	ViewFormats                []TextureFormat
}

// CurrentSurfaceTextureStatus is the outcome of acquiring a surface texture (wgpu::CurrentSurfaceTexture).
type CurrentSurfaceTextureStatus uint8

const (
	CurrentSurfaceTextureSuccess CurrentSurfaceTextureStatus = iota
	CurrentSurfaceTextureSuboptimal
	CurrentSurfaceTextureTimeout
	CurrentSurfaceTextureOccluded
	CurrentSurfaceTextureOutdated
	CurrentSurfaceTextureLost
	CurrentSurfaceTextureValidation
)

// CurrentSurfaceTexture is an acquired surface texture; Texture is set for Success and Suboptimal.
type CurrentSurfaceTexture struct {
	Status  CurrentSurfaceTextureStatus
	Texture Texture
}

// SubmissionIndex identifies a queue submission.
type SubmissionIndex uint64

// Instance creates surfaces and finds adapters.
type Instance interface {
	EnumerateAdapters(backends Backends) []Adapter
	RequestAdapter(options *RequestAdapterOptions) (Adapter, error)
	CreateSurface(target SurfaceTarget) (Surface, error)
	Release()
}

// Adapter is a physical GPU.
type Adapter interface {
	GetInfo() AdapterInfo
	Features() Features
	Limits() Limits
	GetTextureFormatFeatures(format TextureFormat) TextureFormatFeatures
	GetDownlevelCapabilities() DownlevelCapabilities
	RequestDevice(descriptor *DeviceDescriptor) (Device, Queue, error)
	Release()
}

// Device creates GPU resources. Validation errors of its calls go to the innermost error
// scope, else to the OnUncapturedError handler, as in wgpu.
type Device interface {
	Features() Features
	Limits() Limits
	SetDeviceLostCallback(callback func(reason DeviceLostReason, message string))
	OnUncapturedError(handler func(err *Error))
	PushErrorScope(filter ErrorFilter)
	PopErrorScope() *Error
	CreateTexture(descriptor *TextureDescriptor) Texture
	CreateBuffer(descriptor *BufferDescriptor) Buffer
	CreateSampler(descriptor *SamplerDescriptor) Sampler
	CreateShaderModule(descriptor *ShaderModuleDescriptor) ShaderModule
	CreateBindGroupLayout(descriptor *BindGroupLayoutDescriptor) BindGroupLayout
	CreateBindGroup(descriptor *BindGroupDescriptor) BindGroup
	CreatePipelineLayout(descriptor *PipelineLayoutDescriptor) PipelineLayout
	CreateRenderPipeline(descriptor *RenderPipelineDescriptor) RenderPipeline
	CreateCommandEncoder(descriptor *CommandEncoderDescriptor) CommandEncoder
	Poll(wait bool) error
	Release()
}

// Queue submits work and writes resources.
type Queue interface {
	WriteBuffer(buffer Buffer, offset uint64, data []byte)
	WriteTexture(texture TexelCopyTextureInfo, data []byte, layout TexelCopyBufferLayout, size Extent3d)
	Submit(commands ...CommandBuffer) SubmissionIndex
	Release()
}

// Surface is a window's presentable surface.
type Surface interface {
	GetCapabilities(adapter Adapter) SurfaceCapabilities
	Configure(device Device, config *SurfaceConfiguration)
	GetCurrentTexture() CurrentSurfaceTexture
	Present()
	Release()
}

// Texture is a GPU texture.
type Texture interface {
	CreateView(descriptor *TextureViewDescriptor) TextureView
	Width() uint32
	Height() uint32
	Format() TextureFormat
	Destroy()
	Release()
}

// TextureView is a view of a texture.
type TextureView interface {
	Release()
}

// Buffer is a GPU buffer.
type Buffer interface {
	Size() uint64
	MapAsync(mode MapMode, offset, size uint64, callback func(err error))
	GetMappedRange(offset, size uint64) []byte
	Unmap()
	Destroy()
	Release()
}

// Sampler is a texture sampler.
type Sampler interface{ Release() }

// ShaderModule is a compiled shader module.
type ShaderModule interface{ Release() }

// BindGroupLayout is a bind group layout.
type BindGroupLayout interface{ Release() }

// BindGroup is a set of bound resources.
type BindGroup interface{ Release() }

// PipelineLayout is a pipeline layout.
type PipelineLayout interface{ Release() }

// RenderPipeline is a render pipeline.
type RenderPipeline interface{ Release() }

// CommandBuffer is finished, submittable work.
type CommandBuffer interface{ Release() }

// CommandEncoder records GPU commands.
type CommandEncoder interface {
	BeginRenderPass(descriptor *RenderPassDescriptor) RenderPass
	CopyTextureToBuffer(source TexelCopyTextureInfo, destination TexelCopyBufferInfo, size Extent3d)
	Finish() CommandBuffer
	Release()
}

// RenderPass records draws; End finishes it.
type RenderPass interface {
	SetPipeline(pipeline RenderPipeline)
	SetBindGroup(index uint32, group BindGroup, offsets []uint32)
	SetViewport(x, y, width, height, minDepth, maxDepth float32)
	SetScissorRect(x, y, width, height uint32)
	Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32)
	End()
}
