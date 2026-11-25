//go:build windows

package wgc

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/megascan/gpui-go/internal/win/d3d"
)

// ErrFeatureLevelNotSatisfied is d3d11::Error::FeatureLevelNotSatisfied.
var ErrFeatureLevelNotSatisfied = errors.New("Failed to create DirectX device with the recommended feature levels")

// D3D11Error is d3d11::Error::WindowsError.
type D3D11Error struct{ Err error }

func (e *D3D11Error) Error() string { return fmt.Sprintf("Windows API Error: %v", e.Err) }

func (e *D3D11Error) Unwrap() error { return e.Err }

// SendDirectX wraps a DirectX object to move it to another thread.
type SendDirectX[T any] struct{ V T }

// NewSendDirectX wraps device.
func NewSendDirectX[T any](device T) SendDirectX[T] { return SendDirectX[T]{V: device} }

// CreateD3DDevice creates a hardware ID3D11Device with BGRA support and its immediate context.
func CreateD3DDevice() (*d3d.ID3D11Device, *d3d.ID3D11DeviceContext, error) {
	featureFlags := []uint32{
		d3d.D3D_FEATURE_LEVEL_11_1,
		d3d.D3D_FEATURE_LEVEL_11_0,
		d3d.D3D_FEATURE_LEVEL_10_1,
		d3dFeatureLevel10_0,
		d3dFeatureLevel9_3,
		d3dFeatureLevel9_2,
		d3dFeatureLevel9_1,
	}
	device, featureLevel, context, err := d3d.D3D11CreateDevice(nil, d3d.D3D_DRIVER_TYPE_HARDWARE, 0, d3d.D3D11_CREATE_DEVICE_BGRA_SUPPORT, featureFlags, d3d.D3D11_SDK_VERSION)
	if err != nil {
		return nil, nil, &D3D11Error{Err: err}
	}
	if featureLevel < d3d.D3D_FEATURE_LEVEL_11_0 {
		device.Release()
		context.Release()
		return nil, nil, ErrFeatureLevelNotSatisfied
	}
	return device, context, nil
}

// CreateDirect3DDevice wraps an ID3D11Device as a WinRT IDirect3DDevice.
func CreateDirect3DDevice(device *d3d.ID3D11Device) (*IDirect3DDevice, error) {
	dxgiDevice, err := device.CastDXGIDevice()
	if err != nil {
		return nil, &D3D11Error{Err: err}
	}
	defer dxgiDevice.Release()
	inspectable, err := createDirect3D11DeviceFromDXGIDevice(&dxgiDevice.IUnknown)
	if err != nil {
		return nil, &D3D11Error{Err: err}
	}
	defer inspectable.Release()
	out, err := inspectable.QueryInterface(&IID_IDirect3DDevice)
	if err != nil {
		return nil, &D3D11Error{Err: err}
	}
	return (*IDirect3DDevice)(unsafe.Pointer(out)), nil
}
