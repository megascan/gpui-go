//go:build linux

package wgpu

import (
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu/webgpu"
)

func createInstance(descriptor gpu.InstanceDescriptor) gpu.Instance {
	return webgpu.CreateInstance(descriptor)
}

const webGlPrefersRgba = false

const nativeSupportsTransientAttachments = webgpu.SupportsTransientAttachment

func browserSupportsTransientAttachments() bool {
	return false
}
