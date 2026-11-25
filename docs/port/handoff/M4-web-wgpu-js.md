# M4-web-wgpu-js handoff

## For M4-web-platform

- `wgpu.NewWgpuContextWeb(cx scheduler.Awaiter, canvas js.Value, preference wgpu.WebBackendPreference) (*wgpu.PreparedWebGraphics, error)` is upstream's `WgpuContext::new_web(...).await`: call it from a foreground task body with its awaiter. Promises resolve on a helper goroutine; never call it from inside a js.Func callback.
- `PreparedWebGraphics{Context *WgpuContext, Surface gpu.Surface}`; `Release()` drops both. Pass them to `NewWgpuRendererFromSurface`. `NewWgpuRendererFromCanvas(context, canvas, config)` creates another surface on the same instance.
- `WebBackendPreference` constants: `WebBackendPreferenceAuto` (zero value, upstream Default), `WebBackendPreferenceWebGpu`, `WebBackendPreferenceWebGl`.
- The `is_browser_webgpu_supported` probe used in upstream platform.rs maps to `navigator.gpu` presence plus a successful adapter request (an error from `NewWgpuContextWeb` with preference WebGpu).

## For M4-web-webgl

- Replace `wgpuContextNewWebGl` in render/wgpu/wgpu_context_js.go (and `errWebGlUnavailable`, plus its test `TestWgpuContextJS_WebGlBranchReturnsError`) with the webgl2 context; set row `WgpuContext::new_web` to `done` and remove D-WEBGLPENDING from parity/wgpu.md.
- The webgpu facade on js reports every adapter as `gpu.BackendBrowserWebGpu` (the binding's js `GetInfo` is empty).
