# Handoff: M4-apple-metal-atlas

For M4-apple-metal-renderer:
- `stitchedShaderSource()` (platform/apple/shaders.go) is the runtime_shaders source; compile with `device.NewLibraryWithSource(src, mtl.NewCompileOptions())`. `shaderFunctionNames` lists the 9 pipeline vertex/fragment pairs in build order.
- `NewMetalAtlas(device, supportsSharedStorage)` retains the device; `MetalTexture(id)` returns a +1 texture the caller releases; `(*MetalAtlas).Release` drops everything.
- `SurfaceBounds` (metal_renderer.rs) is not ported; internal/cmd/metalhdr/main_test.go checks the header against a test-local `surfaceBounds` struct. When the renderer adds `SurfaceBounds`, the test cannot import package apple (darwin); keep the local mirror in sync.
- The *InputIndex enums in scene.h are skipped by the layout test (`rendererEnums`); the renderer should add its own value check.
