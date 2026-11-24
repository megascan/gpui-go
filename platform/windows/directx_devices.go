//go:build windows

package windows

import (
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unsafe"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/d3d"
)

func tryToRecoverFromDeviceLost[T any](f func() (T, error)) (T, error) {
	var result T
	var err error
	for i := range 5 {
		if i > 0 {
			time.Sleep(time.Duration(100+i*10) * time.Millisecond)
		}
		result, err = f()
		if err == nil {
			return result, nil
		}
	}
	return result, fmt.Errorf("DirectXRenderer failed to recover from lost device after multiple attempts: %w", err)
}

// DirectXDevices holds the DXGI factory, the adapter and the Direct3D 11 device and immediate
// context the renderer draws with. Each field is one counted reference.
type DirectXDevices struct {
	Adapter       *d3d.IDXGIAdapter1
	DxgiFactory   *d3d.IDXGIFactory6
	Device        *d3d.ID3D11Device
	DeviceContext *d3d.ID3D11DeviceContext
}

// NewDirectXDevices creates the devices on the first adapter that supports Direct3D 11 with
// structured buffers.
func NewDirectXDevices() (*DirectXDevices, error) {
	debugLayerAvailable := checkDebugLayerAvailable()
	dxgiFactory, err := getDxgiFactory(debugLayerAvailable)
	if err != nil {
		return nil, fmt.Errorf("Creating DXGI factory: %w", err)
	}
	adapter, device, deviceContext, featureLevel, err := getAdapter(dxgiFactory, debugLayerAvailable)
	if err != nil {
		dxgiFactory.Release()
		return nil, fmt.Errorf("Getting DXGI adapter: %w", err)
	}
	switch featureLevel {
	case d3d.D3D_FEATURE_LEVEL_11_1:
		slog.Info("Created device with Direct3D 11.1 feature level.")
	case d3d.D3D_FEATURE_LEVEL_11_0:
		slog.Info("Created device with Direct3D 11.0 feature level.")
	case d3d.D3D_FEATURE_LEVEL_10_1:
		slog.Info("Created device with Direct3D 10.1 feature level.")
	default:
		panic("internal error: entered unreachable code")
	}
	return &DirectXDevices{
		Adapter:       adapter,
		DxgiFactory:   dxgiFactory,
		Device:        device,
		DeviceContext: deviceContext,
	}, nil
}

// Clone returns a copy holding its own references, as cloning the Rust struct does.
func (d *DirectXDevices) Clone() *DirectXDevices {
	return &DirectXDevices{
		Adapter:       d.Adapter.Clone(),
		DxgiFactory:   d.DxgiFactory.Clone(),
		Device:        d.Device.Clone(),
		DeviceContext: d.DeviceContext.Clone(),
	}
}

// Release drops the references, as dropping the Rust struct does. It is idempotent.
func (d *DirectXDevices) Release() {
	if d.Device == nil {
		return
	}
	d.DeviceContext.Release()
	d.Device.Release()
	d.DxgiFactory.Release()
	d.Adapter.Release()
	*d = DirectXDevices{}
}

func checkDebugLayerAvailable() bool {
	if !gpuiutil.DebugAssertions {
		return false
	}
	queue, ok := gpuiutil.Result(d3d.DXGIGetDebugInterface1(0)).LogErr()
	if !ok {
		return false
	}
	queue.Release()
	return true
}

func getDxgiFactory(debugLayerAvailable bool) (*d3d.IDXGIFactory6, error) {
	var factoryFlag uint32
	if debugLayerAvailable {
		factoryFlag = d3d.DXGI_CREATE_FACTORY_DEBUG
	} else if gpuiutil.DebugAssertions {
		slog.Warn("Failed to get DXGI debug interface. DirectX debugging features will be disabled.")
	}
	return d3d.CreateDXGIFactory2(factoryFlag)
}

func getAdapter(dxgiFactory *d3d.IDXGIFactory6, debugLayerAvailable bool) (*d3d.IDXGIAdapter1, *d3d.ID3D11Device, *d3d.ID3D11DeviceContext, uint32, error) {
	for adapterIndex := uint32(0); ; adapterIndex++ {
		base, err := dxgiFactory.EnumAdapters(adapterIndex)
		if err != nil {
			return nil, nil, nil, 0, err
		}
		adapter, err := base.CastAdapter1()
		base.Release()
		if err != nil {
			return nil, nil, nil, 0, err
		}
		if desc, err := adapter.GetDesc1(); err == nil {
			slog.Info(fmt.Sprintf("Using GPU: %s", desc.DescriptionString()))
		}
		device, context, featureLevel, err := getDevice(adapter, debugLayerAvailable)
		if _, ok := gpuiutil.Result(device, err).LogErr(); ok {
			return adapter, device, context, featureLevel, nil
		}
		adapter.Release()
	}
}

func getDevice(adapter *d3d.IDXGIAdapter1, debugLayerAvailable bool) (*d3d.ID3D11Device, *d3d.ID3D11DeviceContext, uint32, error) {
	deviceFlags := d3d.D3D11_CREATE_DEVICE_BGRA_SUPPORT
	if debugLayerAvailable {
		deviceFlags |= d3d.D3D11_CREATE_DEVICE_DEBUG
	}
	device, featureLevel, context, err := d3d.D3D11CreateDevice(
		adapter,
		d3d.D3D_DRIVER_TYPE_UNKNOWN,
		0,
		deviceFlags,
		[]uint32{
			d3d.D3D_FEATURE_LEVEL_11_1,
			d3d.D3D_FEATURE_LEVEL_11_0,
			d3d.D3D_FEATURE_LEVEL_10_1,
		},
		d3d.D3D11_SDK_VERSION,
	)
	if err != nil {
		return nil, nil, 0, err
	}
	var data d3d.D3D11_FEATURE_DATA_D3D10_X_HARDWARE_OPTIONS
	if err := device.CheckFeatureSupport(d3d.D3D11_FEATURE_D3D10_X_HARDWARE_OPTIONS, unsafe.Pointer(&data), uint32(unsafe.Sizeof(data))); err != nil {
		context.Release()
		device.Release()
		return nil, nil, 0, fmt.Errorf("Checking GPU device feature support: %w", err)
	}
	if data.ComputeShaders_Plus_RawAndStructuredBuffers_Via_Shader_4_x.AsBool() {
		return device, context, featureLevel, nil
	}
	context.Release()
	device.Release()
	return nil, nil, 0, errors.New("Required feature StructuredBuffer is not supported by GPU/driver")
}
