# M4-web-webgl-shaders debt

- naga-cli has no `binding_map` option; ES 3.00 has no `layout(binding)` anyway, so wgpu's GL backend binds by name. glslgen reflects block/sampler names from the GLSL text with regexes and pairs textures with samplers from WGSL `textureSample*` calls. If naga's naming changes, the regexes in internal/cmd/glslgen/main.go must follow.
- Subpixel (dual_source_blending) variant is not generated: WebGL2 has no dual-source blending, so upstream never builds it on the GL path.
- Installed naga-cli reports 30.0.0 (wgpu 30 line).
- A broken mise `naga` shim sits on PATH; glslgen tries $NAGA, ~/.cargo/bin/naga, then PATH, keeping the first that runs `--version`.
