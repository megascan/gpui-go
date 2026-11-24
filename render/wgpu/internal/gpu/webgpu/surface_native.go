//go:build linux

package webgpu

import (
	"errors"
	"unsafe"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/oliverbestmann/webgpu/wgpu"
)

func surfaceDescriptor(target gpu.SurfaceTarget) (*wgpu.SurfaceDescriptor, error) {
	desc := &wgpu.SurfaceDescriptor{}
	switch {
	case target.WaylandSurface != 0:
		desc.WaylandSurface = &wgpu.SurfaceSourceWaylandSurface{
			Display: nativePointer(&target.WaylandDisplay),
			Surface: nativePointer(&target.WaylandSurface),
		}
	case target.XcbWindow != 0:
		desc.XcbWindow = &wgpu.SurfaceSourceXcbWindow{
			Connection: nativePointer(&target.XcbConnection),
			Window:     target.XcbWindow,
		}
	case target.XlibWindow != 0:
		desc.XlibWindow = &wgpu.SurfaceSourceXlibWindow{
			Display: nativePointer(&target.XlibDisplay),
			Window:  target.XlibWindow,
		}
	default:
		return nil, errors.New("unsupported surface target")
	}
	return desc, nil
}

func nativePointer(handle *uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(handle))
}

func adapterBackend(info wgpu.AdapterInfo) gpu.Backend {
	return backend(info.BackendType)
}
