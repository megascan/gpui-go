//go:build !gpui_release

package gpuiutil

// DebugAssertions mirrors cfg!(debug_assertions); build with -tags gpui_release to turn it off.
const DebugAssertions = true
