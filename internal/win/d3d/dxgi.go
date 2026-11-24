//go:build windows

package d3d

import (
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/megascan/gpui-go/internal/win/com"
)

var (
	modDXGI                    = windows.NewLazySystemDLL("dxgi.dll")
	procCreateDXGIFactory2     = modDXGI.NewProc("CreateDXGIFactory2")
	procDXGIGetDebugInterface1 = modDXGI.NewProc("DXGIGetDebugInterface1")
)

const dxgiObjectSlots = com.IUnknownSlots + 4

// CreateDXGIFactory2 creates an IDXGIFactory6.
func CreateDXGIFactory2(flags uint32) (*IDXGIFactory6, error) {
	if err := procCreateDXGIFactory2.Find(); err != nil {
		return nil, err
	}
	var out *IDXGIFactory6
	r, _, _ := procCreateDXGIFactory2.Call(uintptr(flags), uintptr(unsafe.Pointer(&IID_IDXGIFactory6)), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// DXGIGetDebugInterface1 returns the DXGI info queue, which exists only when the graphics tools
// (debug layer) are installed.
func DXGIGetDebugInterface1(flags uint32) (*IDXGIInfoQueue, error) {
	if err := procDXGIGetDebugInterface1.Find(); err != nil {
		return nil, err
	}
	var out *IDXGIInfoQueue
	r, _, _ := procDXGIGetDebugInterface1.Call(uintptr(flags), uintptr(unsafe.Pointer(&IID_IDXGIInfoQueue)), uintptr(unsafe.Pointer(&out)))
	if err := com.HRESULTFromUintptr(r).Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// IDXGIInfoQueue is the DXGI debug message queue.
type IDXGIInfoQueue struct{ com.IUnknown }

// IDXGIFactory6 is a DXGI factory.
type IDXGIFactory6 struct{ com.IUnknown }

// Clone adds a reference and returns f.
func (f *IDXGIFactory6) Clone() *IDXGIFactory6 {
	f.AddRef()
	return f
}

// EnumAdapters returns adapter index; DXGI_ERROR_NOT_FOUND past the last one.
func (f *IDXGIFactory6) EnumAdapters(index uint32) (*IDXGIAdapter, error) {
	var out *IDXGIAdapter
	err := com.CallHR(f.Ptr(), dxgiObjectSlots, uintptr(index), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// MakeWindowAssociation sets how DXGI responds to window messages for hwnd.
func (f *IDXGIFactory6) MakeWindowAssociation(hwnd windows.HWND, flags uint32) error {
	return com.CallHR(f.Ptr(), dxgiObjectSlots+1, uintptr(hwnd), uintptr(flags))
}

// EnumAdapters1 returns adapter index as an IDXGIAdapter1.
func (f *IDXGIFactory6) EnumAdapters1(index uint32) (*IDXGIAdapter1, error) {
	var out *IDXGIAdapter1
	err := com.CallHR(f.Ptr(), dxgiObjectSlots+5, uintptr(index), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateSwapChainForHwnd creates a swap chain for a window.
func (f *IDXGIFactory6) CreateSwapChainForHwnd(device *com.IUnknown, hwnd windows.HWND, desc *DXGI_SWAP_CHAIN_DESC1) (*IDXGISwapChain1, error) {
	var out *IDXGISwapChain1
	err := com.CallHR(f.Ptr(), dxgiObjectSlots+8, iptr(device), uintptr(hwnd), uintptr(unsafe.Pointer(desc)), 0, 0, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// CreateSwapChainForComposition creates a swap chain for DirectComposition.
func (f *IDXGIFactory6) CreateSwapChainForComposition(device *com.IUnknown, desc *DXGI_SWAP_CHAIN_DESC1) (*IDXGISwapChain1, error) {
	var out *IDXGISwapChain1
	err := com.CallHR(f.Ptr(), dxgiObjectSlots+17, iptr(device), uintptr(unsafe.Pointer(desc)), 0, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDXGIAdapter is a display adapter.
type IDXGIAdapter struct{ com.IUnknown }

// CastAdapter1 queries the adapter for IDXGIAdapter1.
func (a *IDXGIAdapter) CastAdapter1() (*IDXGIAdapter1, error) {
	return com.Cast[IDXGIAdapter1](&a.IUnknown, &IID_IDXGIAdapter1)
}

// IDXGIAdapter1 is a display adapter with flags.
type IDXGIAdapter1 struct{ com.IUnknown }

// Clone adds a reference and returns a.
func (a *IDXGIAdapter1) Clone() *IDXGIAdapter1 {
	a.AddRef()
	return a
}

// GetDesc returns the adapter description.
func (a *IDXGIAdapter1) GetDesc() (DXGI_ADAPTER_DESC, error) {
	var desc DXGI_ADAPTER_DESC
	err := com.CallHR(a.Ptr(), dxgiObjectSlots+1, uintptr(unsafe.Pointer(&desc)))
	return desc, err
}

// CheckInterfaceSupport reports whether the adapter supports interface iid and returns the user
// mode driver version.
func (a *IDXGIAdapter1) CheckInterfaceSupport(iid *com.GUID) (int64, error) {
	var umdVersion int64
	err := com.CallHR(a.Ptr(), dxgiObjectSlots+2, uintptr(unsafe.Pointer(iid)), uintptr(unsafe.Pointer(&umdVersion)))
	return umdVersion, err
}

// GetDesc1 returns the adapter description with flags.
func (a *IDXGIAdapter1) GetDesc1() (DXGI_ADAPTER_DESC1, error) {
	var desc DXGI_ADAPTER_DESC1
	err := com.CallHR(a.Ptr(), dxgiObjectSlots+3, uintptr(unsafe.Pointer(&desc)))
	return desc, err
}

// IDXGIDevice is the DXGI side of a Direct3D device.
type IDXGIDevice struct{ com.IUnknown }

// GetAdapter returns the device's adapter.
func (d *IDXGIDevice) GetAdapter() (*IDXGIAdapter, error) {
	var out *IDXGIAdapter
	err := com.CallHR(d.Ptr(), dxgiObjectSlots, uintptr(unsafe.Pointer(&out)))
	return out, err
}

// IDXGISwapChain1 is a swap chain.
type IDXGISwapChain1 struct{ com.IUnknown }

const swapChainSlots = dxgiObjectSlots + 1

// Present presents the back buffer. It returns the raw HRESULT because success codes such as
// DXGI_STATUS_OCCLUDED are meaningful.
func (s *IDXGISwapChain1) Present(syncInterval, flags uint32) com.HRESULT {
	return com.HRESULTFromUintptr(com.Call(s.Ptr(), swapChainSlots, uintptr(syncInterval), uintptr(flags)))
}

// GetBuffer returns back buffer index as a texture.
func (s *IDXGISwapChain1) GetBuffer(index uint32) (*ID3D11Texture2D, error) {
	var out *ID3D11Texture2D
	err := com.CallHR(s.Ptr(), swapChainSlots+1, uintptr(index), uintptr(unsafe.Pointer(&IID_ID3D11Texture2D)), uintptr(unsafe.Pointer(&out)))
	return out, err
}

// ResizeBuffers resizes the swap chain's buffers.
func (s *IDXGISwapChain1) ResizeBuffers(bufferCount, width, height, format, flags uint32) error {
	return com.CallHR(s.Ptr(), swapChainSlots+5, uintptr(bufferCount), uintptr(width), uintptr(height), uintptr(format), uintptr(flags))
}

// GetDesc1 returns the swap chain description.
func (s *IDXGISwapChain1) GetDesc1() (DXGI_SWAP_CHAIN_DESC1, error) {
	var desc DXGI_SWAP_CHAIN_DESC1
	err := com.CallHR(s.Ptr(), swapChainSlots+10, uintptr(unsafe.Pointer(&desc)))
	return desc, err
}
