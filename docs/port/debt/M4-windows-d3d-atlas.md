# M4-windows-d3d-atlas debt

- Windows tests (internal/win/d3d, directx_atlas_test.go, directx_shaders_test.go) were only cross-compiled and vetted on Linux; first real run is M4-ci-windows. Vtable slots come from windows-rs 0.61.3 metadata; struct sizes are pinned in TestD3d_LayoutSizes.
- `ID3D11DeviceContext.ClearDepthStencilView` goes through purego (as dwrite's float calls) because SyscallN cannot pass a by-value float on windows/arm64; any further by-value float method added to internal/win/d3d must do the same.
- The shader `d3d.Include` (alpha_correction.hlsl in native memory) and its 2-slot vtable live for the process; compiled bytecode is cached per module/target in `compiledShaders`.
- `d3d.ID3D11DeviceContext1.DiscardView` takes `*com.IUnknown`; callers pass `com.Unknown(view)`.
