//go:build js

package wgpu

import (
	"syscall/js"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu/webgpu"
)

func createInstance(descriptor gpu.InstanceDescriptor) gpu.Instance {
	return webgpu.CreateInstance(descriptor)
}

const webGlPrefersRgba = true

const nativeSupportsTransientAttachments = true

func browserSupportsTransientAttachments() bool {
	usage := js.Global().Get("GPUTextureUsage")
	if usage.Type() != js.TypeObject && usage.Type() != js.TypeFunction {
		return false
	}
	return js.Global().Get("Reflect").Call("has", usage, "TRANSIENT_ATTACHMENT").Bool()
}
