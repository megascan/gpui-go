//go:build js

package webgpu

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/oliverbestmann/webgpu/wgpu"
)

// BrowserWebGpuSupported reports whether navigator.gpu exists.
func BrowserWebGpuSupported() bool {
	navigator := js.Global().Get("navigator")
	return navigator.Truthy() && navigator.Get("gpu").Truthy()
}

// CreateCanvasSurface creates a surface that presents to an HTML canvas element.
func CreateCanvasSurface(inst gpu.Instance, canvas js.Value) (gpu.Surface, error) {
	i, ok := inst.(*instance)
	if !ok || i.raw == nil {
		return nil, errors.New("browser WebGPU is not available")
	}
	if !canvas.Truthy() {
		return nil, errors.New("canvas element is undefined")
	}
	return i.createSurface(&wgpu.SurfaceDescriptor{Canvas: canvas})
}

func surfaceDescriptor(target gpu.SurfaceTarget) (*wgpu.SurfaceDescriptor, error) {
	if target.CanvasID == "" {
		return nil, errors.New("unsupported surface target")
	}
	canvas := js.Global().Get("document").Call("getElementById", target.CanvasID)
	if !canvas.Truthy() {
		return nil, fmt.Errorf("no canvas element with id %q", target.CanvasID)
	}
	return &wgpu.SurfaceDescriptor{Canvas: canvas}, nil
}

func adapterBackend(wgpu.AdapterInfo) gpu.Backend {
	return gpu.BackendBrowserWebGpu
}
