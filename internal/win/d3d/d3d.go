//go:build windows

// Package d3d binds the Direct3D 11, DXGI, DirectComposition and D3DCompiler interfaces gpui's
// Windows renderer, atlas and screen capture use.
package d3d

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

// Interface ids.
var (
	IID_ID3D11Device              = com.MustGUID("{DB6F6DDB-AC77-4E88-8253-819DF9BBF140}")
	IID_ID3D11DeviceContext       = com.MustGUID("{C0BFA96C-E089-44FB-8EAF-26F8796190DA}")
	IID_ID3D11DeviceContext1      = com.MustGUID("{BB2C6FAA-B5FB-4082-8E6B-388B8CFA90E1}")
	IID_ID3D11Resource            = com.MustGUID("{DC8E63F3-D12B-4952-B47B-5E45026A862D}")
	IID_ID3D11Texture2D           = com.MustGUID("{6F15AAF2-D208-4E89-9AB4-489535D34F9C}")
	IID_ID3D11Buffer              = com.MustGUID("{48570B85-D1EE-4FCD-A250-EB350722B037}")
	IID_ID3D11ShaderResourceView  = com.MustGUID("{B0E06FE0-8192-4E1A-B1CA-36D7414710B2}")
	IID_ID3D11RenderTargetView    = com.MustGUID("{DFDBA067-0B8D-4865-875B-D7B4516CC164}")
	IID_ID3D11DepthStencilView    = com.MustGUID("{9FDAC92A-1876-48C3-AFAD-25B94F84A9B6}")
	IID_ID3D11Debug               = com.MustGUID("{79CF2233-7536-4948-9D36-1E4692DC5760}")
	IID_ID3DUserDefinedAnnotation = com.MustGUID("{B2DAAD8B-03D4-4DBF-95EB-32AB4B63D0AB}")
	IID_ID3DBlob                  = com.MustGUID("{8BA5FB08-5195-40E2-AC58-0D989C3A0102}")
	IID_IDXGIFactory6             = com.MustGUID("{C1B6694F-FF09-44A9-B03C-77900A0A1D17}")
	IID_IDXGIAdapter              = com.MustGUID("{2411E7E1-12AC-4CCF-BD14-9798E8534DC0}")
	IID_IDXGIAdapter1             = com.MustGUID("{29038F61-3839-4626-91FD-086879011A05}")
	IID_IDXGIDevice               = com.MustGUID("{54EC77FA-1377-44E6-8C32-88FD5F44C84C}")
	IID_IDXGISwapChain1           = com.MustGUID("{790A45F7-0D42-4876-983A-0A55CFE6F4AA}")
	IID_IDXGIInfoQueue            = com.MustGUID("{D67441C7-672A-476F-9E82-CD55B44949CE}")
	IID_IDCompositionDevice       = com.MustGUID("{C37EA93A-E7AA-450D-B16F-9746CB0407F3}")
	IID_IDCompositionTarget       = com.MustGUID("{EACDD04C-117E-4E17-88F4-D1B12B0E3D89}")
	IID_IDCompositionVisual       = com.MustGUID("{4D93059D-097B-4651-9A60-F0F25116E2F3}")
	DXGI_DEBUG_ALL                = com.MustGUID("{E48AE283-DA80-490B-87E6-43E9A9CFDA08}")
)

// DXGI HRESULTs.
const (
	DXGI_ERROR_INVALID_CALL          com.HRESULT = -0x7785FFFF
	DXGI_ERROR_NOT_FOUND             com.HRESULT = -0x7785FFFE
	DXGI_ERROR_DEVICE_REMOVED        com.HRESULT = -0x7785FFFB
	DXGI_ERROR_DEVICE_HUNG           com.HRESULT = -0x7785FFFA
	DXGI_ERROR_DEVICE_RESET          com.HRESULT = -0x7785FFF9
	DXGI_ERROR_DRIVER_INTERNAL_ERROR com.HRESULT = -0x7785FFE0
	DXGI_ERROR_UNSUPPORTED           com.HRESULT = -0x7785FFFC
	DXGI_STATUS_OCCLUDED             com.HRESULT = 0x087A0001
)

// Enumerations and flags, as their C values.
const (
	D3D_DRIVER_TYPE_UNKNOWN  uint32 = 0
	D3D_DRIVER_TYPE_HARDWARE uint32 = 1
	D3D_DRIVER_TYPE_WARP     uint32 = 5

	D3D_FEATURE_LEVEL_10_1 uint32 = 0xa100
	D3D_FEATURE_LEVEL_11_0 uint32 = 0xb000
	D3D_FEATURE_LEVEL_11_1 uint32 = 0xb100

	D3D_PRIMITIVE_TOPOLOGY_TRIANGLELIST  uint32 = 4
	D3D_PRIMITIVE_TOPOLOGY_TRIANGLESTRIP uint32 = 5

	D3D11_SDK_VERSION uint32 = 7

	D3D11_CREATE_DEVICE_SINGLETHREADED uint32 = 0x1
	D3D11_CREATE_DEVICE_DEBUG          uint32 = 0x2
	D3D11_CREATE_DEVICE_BGRA_SUPPORT   uint32 = 0x20

	D3D11_FEATURE_D3D10_X_HARDWARE_OPTIONS uint32 = 4

	D3D11_BIND_VERTEX_BUFFER   uint32 = 0x1
	D3D11_BIND_INDEX_BUFFER    uint32 = 0x2
	D3D11_BIND_CONSTANT_BUFFER uint32 = 0x4
	D3D11_BIND_SHADER_RESOURCE uint32 = 0x8
	D3D11_BIND_RENDER_TARGET   uint32 = 0x20
	D3D11_BIND_DEPTH_STENCIL   uint32 = 0x40

	D3D11_USAGE_DEFAULT   uint32 = 0
	D3D11_USAGE_IMMUTABLE uint32 = 1
	D3D11_USAGE_DYNAMIC   uint32 = 2
	D3D11_USAGE_STAGING   uint32 = 3

	D3D11_CPU_ACCESS_WRITE uint32 = 0x10000
	D3D11_CPU_ACCESS_READ  uint32 = 0x20000

	D3D11_RESOURCE_MISC_BUFFER_STRUCTURED uint32 = 0x40

	D3D11_MAP_READ          uint32 = 1
	D3D11_MAP_WRITE         uint32 = 2
	D3D11_MAP_READ_WRITE    uint32 = 3
	D3D11_MAP_WRITE_DISCARD uint32 = 4

	D3D11_CLEAR_DEPTH   uint32 = 0x1
	D3D11_CLEAR_STENCIL uint32 = 0x2

	D3D11_BLEND_ZERO           int32 = 1
	D3D11_BLEND_ONE            int32 = 2
	D3D11_BLEND_SRC_COLOR      int32 = 3
	D3D11_BLEND_INV_SRC_COLOR  int32 = 4
	D3D11_BLEND_SRC_ALPHA      int32 = 5
	D3D11_BLEND_INV_SRC_ALPHA  int32 = 6
	D3D11_BLEND_DEST_ALPHA     int32 = 7
	D3D11_BLEND_INV_DEST_ALPHA int32 = 8
	D3D11_BLEND_SRC1_COLOR     int32 = 16
	D3D11_BLEND_INV_SRC1_COLOR int32 = 17
	D3D11_BLEND_SRC1_ALPHA     int32 = 18
	D3D11_BLEND_INV_SRC1_ALPHA int32 = 19

	D3D11_BLEND_OP_ADD int32 = 1

	D3D11_COLOR_WRITE_ENABLE_RED   uint8 = 1
	D3D11_COLOR_WRITE_ENABLE_GREEN uint8 = 2
	D3D11_COLOR_WRITE_ENABLE_BLUE  uint8 = 4
	D3D11_COLOR_WRITE_ENABLE_ALPHA uint8 = 8
	D3D11_COLOR_WRITE_ENABLE_ALL   uint8 = 15

	D3D11_COMPARISON_NEVER         int32 = 1
	D3D11_COMPARISON_LESS          int32 = 2
	D3D11_COMPARISON_EQUAL         int32 = 3
	D3D11_COMPARISON_LESS_EQUAL    int32 = 4
	D3D11_COMPARISON_GREATER       int32 = 5
	D3D11_COMPARISON_NOT_EQUAL     int32 = 6
	D3D11_COMPARISON_GREATER_EQUAL int32 = 7
	D3D11_COMPARISON_ALWAYS        int32 = 8

	D3D11_DEPTH_WRITE_MASK_ZERO int32 = 0
	D3D11_DEPTH_WRITE_MASK_ALL  int32 = 1

	D3D11_STENCIL_OP_KEEP int32 = 1

	D3D11_DEFAULT_STENCIL_READ_MASK  uint8 = 0xff
	D3D11_DEFAULT_STENCIL_WRITE_MASK uint8 = 0xff

	D3D11_FILL_WIREFRAME int32 = 2
	D3D11_FILL_SOLID     int32 = 3

	D3D11_CULL_NONE  int32 = 1
	D3D11_CULL_FRONT int32 = 2
	D3D11_CULL_BACK  int32 = 3

	D3D11_FILTER_MIN_MAG_MIP_POINT  int32 = 0
	D3D11_FILTER_MIN_MAG_MIP_LINEAR int32 = 0x15

	D3D11_TEXTURE_ADDRESS_WRAP  int32 = 1
	D3D11_TEXTURE_ADDRESS_CLAMP int32 = 3

	D3D11_FLOAT32_MAX float32 = 3.402823466e+38

	D3D11_RLDO_SUMMARY         int32 = 1
	D3D11_RLDO_DETAIL          int32 = 2
	D3D11_RLDO_IGNORE_INTERNAL int32 = 4

	DXGI_FORMAT_UNKNOWN        uint32 = 0
	DXGI_FORMAT_R8G8B8A8_UNORM uint32 = 28
	DXGI_FORMAT_D32_FLOAT      uint32 = 40
	DXGI_FORMAT_R8_UNORM       uint32 = 61
	DXGI_FORMAT_A8_UNORM       uint32 = 65
	DXGI_FORMAT_B8G8R8A8_UNORM uint32 = 87

	DXGI_CREATE_FACTORY_DEBUG uint32 = 0x1

	DXGI_ADAPTER_FLAG_SOFTWARE uint32 = 2

	DXGI_USAGE_SHADER_INPUT         uint32 = 0x10
	DXGI_USAGE_RENDER_TARGET_OUTPUT uint32 = 0x20

	DXGI_SCALING_STRETCH uint32 = 0
	DXGI_SCALING_NONE    uint32 = 1

	DXGI_SWAP_EFFECT_DISCARD         uint32 = 0
	DXGI_SWAP_EFFECT_SEQUENTIAL      uint32 = 1
	DXGI_SWAP_EFFECT_FLIP_SEQUENTIAL uint32 = 3
	DXGI_SWAP_EFFECT_FLIP_DISCARD    uint32 = 4

	DXGI_ALPHA_MODE_UNSPECIFIED   uint32 = 0
	DXGI_ALPHA_MODE_PREMULTIPLIED uint32 = 1
	DXGI_ALPHA_MODE_STRAIGHT      uint32 = 2
	DXGI_ALPHA_MODE_IGNORE        uint32 = 3

	DXGI_MWA_NO_WINDOW_CHANGES uint32 = 1
	DXGI_MWA_NO_ALT_ENTER      uint32 = 2

	DXGI_PRESENT_TEST uint32 = 0x1

	D3DCOMPILE_DEBUG               uint32 = 1 << 0
	D3DCOMPILE_SKIP_OPTIMIZATION   uint32 = 1 << 2
	D3DCOMPILE_OPTIMIZATION_LEVEL3 uint32 = 1 << 15

	D3D_INCLUDE_LOCAL  uint32 = 0
	D3D_INCLUDE_SYSTEM uint32 = 1
)

// BOOL is a Win32 BOOL.
type BOOL int32

// AsBool reports whether b is nonzero.
func (b BOOL) AsBool() bool {
	return b != 0
}

// BoolOf converts v to a BOOL.
func BoolOf(v bool) BOOL {
	if v {
		return 1
	}
	return 0
}

// LUID is a locally unique identifier.
type LUID struct {
	LowPart  uint32
	HighPart int32
}

// RECT is a Win32 rectangle.
type RECT struct {
	Left, Top, Right, Bottom int32
}

// DXGI_SAMPLE_DESC is a multisampling description.
type DXGI_SAMPLE_DESC struct {
	Count   uint32
	Quality uint32
}

// DXGI_ADAPTER_DESC describes an adapter.
type DXGI_ADAPTER_DESC struct {
	Description           [128]uint16
	VendorId              uint32
	DeviceId              uint32
	SubSysId              uint32
	Revision              uint32
	DedicatedVideoMemory  uintptr
	DedicatedSystemMemory uintptr
	SharedSystemMemory    uintptr
	AdapterLuid           LUID
}

// DXGI_ADAPTER_DESC1 describes an adapter, with flags.
type DXGI_ADAPTER_DESC1 struct {
	Description           [128]uint16
	VendorId              uint32
	DeviceId              uint32
	SubSysId              uint32
	Revision              uint32
	DedicatedVideoMemory  uintptr
	DedicatedSystemMemory uintptr
	SharedSystemMemory    uintptr
	AdapterLuid           LUID
	Flags                 uint32
}

// DescriptionString returns the adapter description with trailing NULs removed.
func (d *DXGI_ADAPTER_DESC1) DescriptionString() string {
	return windows.UTF16ToString(d.Description[:])
}

// DXGI_SWAP_CHAIN_DESC1 describes a swap chain.
type DXGI_SWAP_CHAIN_DESC1 struct {
	Width       uint32
	Height      uint32
	Format      uint32
	Stereo      BOOL
	SampleDesc  DXGI_SAMPLE_DESC
	BufferUsage uint32
	BufferCount uint32
	Scaling     uint32
	SwapEffect  uint32
	AlphaMode   uint32
	Flags       uint32
}

// D3D11_TEXTURE2D_DESC describes a 2D texture.
type D3D11_TEXTURE2D_DESC struct {
	Width          uint32
	Height         uint32
	MipLevels      uint32
	ArraySize      uint32
	Format         uint32
	SampleDesc     DXGI_SAMPLE_DESC
	Usage          uint32
	BindFlags      uint32
	CPUAccessFlags uint32
	MiscFlags      uint32
}

// D3D11_BUFFER_DESC describes a buffer.
type D3D11_BUFFER_DESC struct {
	ByteWidth           uint32
	Usage               uint32
	BindFlags           uint32
	CPUAccessFlags      uint32
	MiscFlags           uint32
	StructureByteStride uint32
}

// D3D11_SUBRESOURCE_DATA is initial data for a resource.
type D3D11_SUBRESOURCE_DATA struct {
	PSysMem          unsafe.Pointer
	SysMemPitch      uint32
	SysMemSlicePitch uint32
}

// D3D11_MAPPED_SUBRESOURCE is a mapped subresource.
type D3D11_MAPPED_SUBRESOURCE struct {
	PData      unsafe.Pointer
	RowPitch   uint32
	DepthPitch uint32
}

// D3D11_BOX is a 3D box.
type D3D11_BOX struct {
	Left   uint32
	Top    uint32
	Front  uint32
	Right  uint32
	Bottom uint32
	Back   uint32
}

// D3D11_VIEWPORT is a viewport.
type D3D11_VIEWPORT struct {
	TopLeftX float32
	TopLeftY float32
	Width    float32
	Height   float32
	MinDepth float32
	MaxDepth float32
}

// D3D11_RENDER_TARGET_BLEND_DESC is the blend state of one render target.
type D3D11_RENDER_TARGET_BLEND_DESC struct {
	BlendEnable           BOOL
	SrcBlend              int32
	DestBlend             int32
	BlendOp               int32
	SrcBlendAlpha         int32
	DestBlendAlpha        int32
	BlendOpAlpha          int32
	RenderTargetWriteMask uint8
}

// D3D11_BLEND_DESC describes a blend state.
type D3D11_BLEND_DESC struct {
	AlphaToCoverageEnable  BOOL
	IndependentBlendEnable BOOL
	RenderTarget           [8]D3D11_RENDER_TARGET_BLEND_DESC
}

// D3D11_DEPTH_STENCILOP_DESC describes stencil operations.
type D3D11_DEPTH_STENCILOP_DESC struct {
	StencilFailOp      int32
	StencilDepthFailOp int32
	StencilPassOp      int32
	StencilFunc        int32
}

// D3D11_DEPTH_STENCIL_DESC describes a depth-stencil state.
type D3D11_DEPTH_STENCIL_DESC struct {
	DepthEnable      BOOL
	DepthWriteMask   int32
	DepthFunc        int32
	StencilEnable    BOOL
	StencilReadMask  uint8
	StencilWriteMask uint8
	FrontFace        D3D11_DEPTH_STENCILOP_DESC
	BackFace         D3D11_DEPTH_STENCILOP_DESC
}

// D3D11_RASTERIZER_DESC describes a rasterizer state.
type D3D11_RASTERIZER_DESC struct {
	FillMode              int32
	CullMode              int32
	FrontCounterClockwise BOOL
	DepthBias             int32
	DepthBiasClamp        float32
	SlopeScaledDepthBias  float32
	DepthClipEnable       BOOL
	ScissorEnable         BOOL
	MultisampleEnable     BOOL
	AntialiasedLineEnable BOOL
}

// D3D11_SAMPLER_DESC describes a sampler state.
type D3D11_SAMPLER_DESC struct {
	Filter         int32
	AddressU       int32
	AddressV       int32
	AddressW       int32
	MipLODBias     float32
	MaxAnisotropy  uint32
	ComparisonFunc int32
	BorderColor    [4]float32
	MinLOD         float32
	MaxLOD         float32
}

// D3D11_FEATURE_DATA_D3D10_X_HARDWARE_OPTIONS reports compute shader support on 10.x hardware.
type D3D11_FEATURE_DATA_D3D10_X_HARDWARE_OPTIONS struct {
	ComputeShaders_Plus_RawAndStructuredBuffers_Via_Shader_4_x BOOL
}

type fnKey struct {
	addr uintptr
	typ  reflect.Type
}

var floatFns sync.Map

func floatFn[F any](this unsafe.Pointer, slot int) F {
	key := fnKey{addr: com.Method(this, slot), typ: reflect.TypeFor[F]()}
	if v, ok := floatFns.Load(key); ok {
		return v.(F)
	}
	var f F
	purego.RegisterFunc(&f, key.addr)
	v, _ := floatFns.LoadOrStore(key, f)
	return v.(F)
}

func slicePtr[T any](s []T) unsafe.Pointer {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Pointer(&s[0])
}
