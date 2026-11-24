//go:build js

package wgpu

import "github.com/megascan/gpui-go/render/wgpu/internal/gpu"

func createInstance(descriptor gpu.InstanceDescriptor) gpu.Instance {
	panic("gpui: unimplemented createInstance")
}

const webGlPrefersRgba = true

const nativeSupportsTransientAttachments = true

func browserSupportsTransientAttachments() bool {
	panic("gpui: unimplemented browserSupportsTransientAttachments")
}
