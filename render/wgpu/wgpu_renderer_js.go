//go:build js

package wgpu

import (
	"fmt"
	"syscall/js"
)

// NewWgpuRendererFromCanvas creates a renderer presenting to canvas with context's instance.
func NewWgpuRendererFromCanvas(context *WgpuContext, canvas js.Value, config WgpuSurfaceConfig) (*WgpuRenderer, error) {
	surface, err := createCanvasSurface(context.Instance, canvas)
	if err != nil {
		return nil, fmt.Errorf("Failed to create surface: %w", err)
	}
	return NewWgpuRendererFromSurface(context, surface, config)
}
