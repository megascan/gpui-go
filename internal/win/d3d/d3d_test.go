//go:build windows

package d3d

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/com"
)

func TestD3d_LayoutSizes(t *testing.T) {
	cases := []struct {
		name string
		got  uintptr
		want uintptr
	}{
		{"D3D11_TEXTURE2D_DESC", unsafe.Sizeof(D3D11_TEXTURE2D_DESC{}), 44},
		{"D3D11_BUFFER_DESC", unsafe.Sizeof(D3D11_BUFFER_DESC{}), 24},
		{"D3D11_RENDER_TARGET_BLEND_DESC", unsafe.Sizeof(D3D11_RENDER_TARGET_BLEND_DESC{}), 32},
		{"D3D11_BLEND_DESC", unsafe.Sizeof(D3D11_BLEND_DESC{}), 264},
		{"D3D11_DEPTH_STENCIL_DESC", unsafe.Sizeof(D3D11_DEPTH_STENCIL_DESC{}), 52},
		{"D3D11_RASTERIZER_DESC", unsafe.Sizeof(D3D11_RASTERIZER_DESC{}), 40},
		{"D3D11_SAMPLER_DESC", unsafe.Sizeof(D3D11_SAMPLER_DESC{}), 52},
		{"D3D11_VIEWPORT", unsafe.Sizeof(D3D11_VIEWPORT{}), 24},
		{"D3D11_BOX", unsafe.Sizeof(D3D11_BOX{}), 24},
		{"D3D11_MAPPED_SUBRESOURCE", unsafe.Sizeof(D3D11_MAPPED_SUBRESOURCE{}), 16},
		{"D3D11_SUBRESOURCE_DATA", unsafe.Sizeof(D3D11_SUBRESOURCE_DATA{}), 16},
		{"DXGI_SWAP_CHAIN_DESC1", unsafe.Sizeof(DXGI_SWAP_CHAIN_DESC1{}), 48},
		{"DXGI_ADAPTER_DESC", unsafe.Sizeof(DXGI_ADAPTER_DESC{}), 304},
		{"DXGI_ADAPTER_DESC1", unsafe.Sizeof(DXGI_ADAPTER_DESC1{}), 312},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("sizeof(%s) = %d, want %d", c.name, c.got, c.want)
		}
	}
	if off := unsafe.Offsetof(D3D11_DEPTH_STENCIL_DESC{}.FrontFace); off != 20 {
		t.Errorf("D3D11_DEPTH_STENCIL_DESC.FrontFace offset = %d, want 20", off)
	}
}

func TestD3d_HRESULTValues(t *testing.T) {
	cases := map[com.HRESULT]uint32{
		DXGI_ERROR_INVALID_CALL:          0x887A0001,
		DXGI_ERROR_NOT_FOUND:             0x887A0002,
		DXGI_ERROR_UNSUPPORTED:           0x887A0004,
		DXGI_ERROR_DEVICE_REMOVED:        0x887A0005,
		DXGI_ERROR_DEVICE_HUNG:           0x887A0006,
		DXGI_ERROR_DEVICE_RESET:          0x887A0007,
		DXGI_ERROR_DRIVER_INTERNAL_ERROR: 0x887A0020,
	}
	for h, want := range cases {
		if uint32(h) != want {
			t.Errorf("%#x != %#x", uint32(h), want)
		}
	}
}

func createWarpDevice(t *testing.T) (*ID3D11Device, *ID3D11DeviceContext) {
	t.Helper()
	device, _, context, err := D3D11CreateDevice(nil, D3D_DRIVER_TYPE_WARP, 0, D3D11_CREATE_DEVICE_BGRA_SUPPORT, nil, D3D11_SDK_VERSION)
	if err != nil {
		t.Skipf("no WARP device: %v", err)
	}
	t.Cleanup(func() {
		context.Release()
		device.Release()
	})
	return device, context
}

func TestD3d_WarpTextureRoundTrip(t *testing.T) {
	device, context := createWarpDevice(t)
	desc := D3D11_TEXTURE2D_DESC{
		Width: 4, Height: 4, MipLevels: 1, ArraySize: 1,
		Format:     DXGI_FORMAT_R8_UNORM,
		SampleDesc: DXGI_SAMPLE_DESC{Count: 1},
		Usage:      D3D11_USAGE_DEFAULT,
		BindFlags:  D3D11_BIND_SHADER_RESOURCE,
	}
	texture, err := device.CreateTexture2D(&desc, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer texture.Release()
	if got := texture.GetDesc(); got != desc {
		t.Fatalf("GetDesc = %+v, want %+v", got, desc)
	}
	view, err := device.CreateShaderResourceView(texture.Resource())
	if err != nil {
		t.Fatal(err)
	}
	defer view.Release()
	pixels := make([]byte, 4)
	for i := range pixels {
		pixels[i] = byte(10 + i)
	}
	context.UpdateSubresource(texture.Resource(), 0, &D3D11_BOX{Left: 1, Top: 2, Right: 3, Bottom: 4, Back: 1}, unsafe.Pointer(&pixels[0]), 2, 0)

	staging := desc
	staging.Usage = D3D11_USAGE_STAGING
	staging.BindFlags = 0
	staging.CPUAccessFlags = D3D11_CPU_ACCESS_READ
	readback, err := device.CreateTexture2D(&staging, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer readback.Release()
	context.CopyResource(readback.Resource(), texture.Resource())
	mapped, err := context.Map(readback.Resource(), 0, D3D11_MAP_READ, 0)
	if err != nil {
		t.Fatal(err)
	}
	row2 := unsafe.Slice((*byte)(unsafe.Add(mapped.PData, 2*uintptr(mapped.RowPitch))), 4)
	row3 := unsafe.Slice((*byte)(unsafe.Add(mapped.PData, 3*uintptr(mapped.RowPitch))), 4)
	got := []byte{row2[1], row2[2], row3[1], row3[2]}
	context.Unmap(readback.Resource(), 0)
	for i := range got {
		if got[i] != pixels[i] {
			t.Fatalf("read back %v, want %v", got, pixels)
		}
	}
	if err := device.GetDeviceRemovedReason(); err != nil {
		t.Fatalf("device removed: %v", err)
	}
}

func TestD3d_CompileWithInclude(t *testing.T) {
	include := NewInclude(map[string][]byte{"inc.hlsl": []byte("float4 color() { return float4(1, 0, 0, 1); }\n")})
	defer include.Free()
	source := []byte("#include \"inc.hlsl\"\nfloat4 main_ps() : SV_Target { return color(); }\n")
	code, errs, err := D3DCompile(source, "test.hlsl", include, "main_ps", "ps_4_1", D3DCOMPILE_OPTIMIZATION_LEVEL3, 0)
	if err != nil {
		msg := ""
		if errs != nil {
			msg = errs.String()
			errs.Release()
		}
		t.Fatalf("compile failed: %v: %s", err, msg)
	}
	defer code.Release()
	if errs != nil {
		errs.Release()
	}
	if len(code.Bytes()) == 0 {
		t.Fatal("empty bytecode")
	}

	bad := []byte("#include \"missing.hlsl\"\nfloat4 main_ps() : SV_Target { return 0; }\n")
	code, errs, err = D3DCompile(bad, "bad.hlsl", include, "main_ps", "ps_4_1", 0, 0)
	if err == nil {
		code.Release()
		t.Fatal("include of an unknown file compiled")
	}
	if errs == nil {
		t.Fatal("no error blob")
	}
	defer errs.Release()
	if !strings.Contains(errs.String(), "missing.hlsl") {
		t.Fatalf("error message %q does not name the file", errs.String())
	}
}
