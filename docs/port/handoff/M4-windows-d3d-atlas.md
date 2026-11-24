# M4-windows-d3d-atlas handoffs

- M4-windows-directx-renderer: `shader_resources` (directx_renderer.rs:1722-1880) is already ported in platform/windows/directx_shaders.go (`ShaderModule`, `ShaderTarget`, `NewRawShaderBytes`, `AsBytes`); do not port it again in directx_renderer.go. `DirectXDevices` / `NewDirectXDevices` / `tryToRecoverFromDeviceLost` are in directx_devices.go, `DirectXAtlas` (`NewDirectXAtlas`, `GetTextureView` returns an AddRef'd view, `HandleDeviceLost`, `Release`) in directx_atlas.go. internal/win/d3d exposes every D3D11/DXGI/DComp call directx_renderer.rs makes; Present returns the raw HRESULT.
- M4-ci-windows: run `go test ./internal/win/d3d/... ./platform/windows/...` on the runner; they create WARP devices and compile every shaders.hlsl entry point.
