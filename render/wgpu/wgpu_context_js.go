//go:build js

package wgpu

import (
	"errors"
	"fmt"
	"log/slog"
	"syscall/js"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu/webgpu"
	"github.com/megascan/gpui-go/scheduler"
)

// WebBackendPreference selects the browser graphics API.
type WebBackendPreference uint8

const (
	WebBackendPreferenceAuto WebBackendPreference = iota
	WebBackendPreferenceWebGpu
	WebBackendPreferenceWebGl
)

// String formats the preference as upstream's Debug does.
func (p WebBackendPreference) String() string {
	switch p {
	case WebBackendPreferenceWebGpu:
		return "WebGpu"
	case WebBackendPreferenceWebGl:
		return "WebGl"
	}
	return "Auto"
}

func (p WebBackendPreference) backends() gpu.Backends {
	switch p {
	case WebBackendPreferenceWebGpu:
		return gpu.BackendsBrowserWebGpu
	case WebBackendPreferenceWebGl:
		return gpu.BackendsGl
	}
	return gpu.BackendsBrowserWebGpu | gpu.BackendsGl
}

// PreparedWebGraphics is a browser context together with the canvas surface it was created for.
type PreparedWebGraphics struct {
	Context *WgpuContext
	Surface gpu.Surface
}

// Release drops the surface and the context's device.
func (p *PreparedWebGraphics) Release() {
	if p.Surface != nil {
		p.Surface.Release()
		p.Surface = nil
	}
	if p.Context != nil {
		p.Context.Queue.Release()
		p.Context.Device.Release()
		p.Context.Adapter.Release()
		p.Context.Instance.Release()
		p.Context = nil
	}
}

type preparedWebGraphicsResult struct {
	graphics *PreparedWebGraphics
	err      error
}

// NewWgpuContextWeb creates a context and surface for canvas. Adapter and device
// acquisition resolve browser promises on a separate goroutine; cx parks until they do.
func NewWgpuContextWeb(cx scheduler.Awaiter, canvas js.Value, preference WebBackendPreference) (*PreparedWebGraphics, error) {
	tx, rx := scheduler.Oneshot[preparedWebGraphicsResult]()
	go func() {
		graphics, err := wgpuContextNewWebWithBackend(canvas, preference)
		if sendErr := tx.Send(preparedWebGraphicsResult{graphics, err}); sendErr != nil && graphics != nil {
			graphics.Release()
		}
	}()
	result, err := rx.Recv(cx)
	if err != nil {
		return nil, err
	}
	return result.graphics, result.err
}

func wgpuContextNewWebWithBackend(canvas js.Value, preference WebBackendPreference) (*PreparedWebGraphics, error) {
	backends := preference.backends()
	if preference == WebBackendPreferenceAuto && !webgpu.BrowserWebGpuSupported() {
		backends &^= gpu.BackendsBrowserWebGpu
	}
	if backends&gpu.BackendsBrowserWebGpu == 0 {
		return wgpuContextNewWebGl(canvas, preference)
	}
	instance := createInstance(gpu.InstanceDescriptor{Backends: backends})
	surface, err := webgpu.CreateCanvasSurface(instance, canvas)
	if err != nil {
		instance.Release()
		return nil, fmt.Errorf("Failed to create browser graphics surface: %w", err)
	}

	adapter, err := instance.RequestAdapter(&gpu.RequestAdapterOptions{
		PowerPreference:   gpu.PowerPreferenceHighPerformance,
		CompatibleSurface: surface,
	})
	if err != nil {
		surface.Release()
		instance.Release()
		return nil, fmt.Errorf("Failed to request a %s adapter compatible with the canvas: %w", preference, err)
	}
	adapterInfo := adapter.GetInfo()
	var backend WgpuBackend
	switch adapterInfo.Backend {
	case gpu.BackendBrowserWebGpu:
		backend = WgpuBackend{Kind: WgpuBackendBrowserWebGpu}
	case gpu.BackendGl:
		backend = WgpuBackend{Kind: WgpuBackendGl}
	default:
		adapter.Release()
		surface.Release()
		instance.Release()
		return nil, fmt.Errorf("Browser graphics initialization selected unexpected backend %s", adapterInfo.Backend)
	}

	device, queue, dualSourceBlending, colorTextureFormat, err := wgpuContextCreateDevice(adapter)
	if err != nil {
		adapter.Release()
		surface.Release()
		instance.Release()
		return nil, err
	}
	errors := installDeviceErrorState(device)
	slog.Info(fmt.Sprintf(
		"Browser graphics initialized: requested=%s, selected=%s, adapter=%q, limits=%+v, dual_source_blending=%t",
		preference,
		backend,
		adapterInfo.Name,
		device.Limits(),
		dualSourceBlending,
	))

	context := &WgpuContext{
		Instance:           instance,
		Adapter:            adapter,
		Device:             device,
		Queue:              queue,
		backend:            backend,
		dualSourceBlending: dualSourceBlending,
		colorTextureFormat: colorTextureFormat,
		errors:             errors,
	}
	return &PreparedWebGraphics{Context: context, Surface: surface}, nil
}

var errWebGlUnavailable = errors.New("the WebGL2 backend is not available in this build")

func wgpuContextNewWebGl(canvas js.Value, preference WebBackendPreference) (*PreparedWebGraphics, error) {
	return nil, fmt.Errorf("Failed to request a %s adapter compatible with the canvas: %w", preference, errWebGlUnavailable)
}
