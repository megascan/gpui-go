# Debt: M4-web-webgl

- render/wgpu/internal/gpu/webgl2/jsgl.go: `getUniformLocation` stores every WebGLUniformLocation in the handle table and never removes it. Each pipeline creation adds a few entries. That is harmless while pipelines are created once per renderer, but the table grows if pipelines are rebuilt many times, for example on repeated device recovery.
