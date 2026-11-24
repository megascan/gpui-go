//go:build linux || js

// Package wgpu is GPUI's wgpu renderer, atlas and GPU context (gpui_wgpu), written against the
// GPU facade in render/wgpu/internal/gpu.
package wgpu

import "github.com/megascan/gpui-go/render/wgpu/internal/gpu"

// PresentMode is a surface presentation mode (wgpu::PresentMode, re-exported as upstream's
// `pub use wgpu`).
type PresentMode = gpu.PresentMode

const (
	PresentModeFifo        = gpu.PresentModeFifo
	PresentModeFifoRelaxed = gpu.PresentModeFifoRelaxed
	PresentModeImmediate   = gpu.PresentModeImmediate
	PresentModeMailbox     = gpu.PresentModeMailbox
)
