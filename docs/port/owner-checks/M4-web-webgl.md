# Owner checks: M4-web-webgl

- YOU: open the web example in a browser with WebGPU disabled (Chrome `--disable-features=WebGPU`, or Firefox without `dom.webgpu.enabled`) or with the backend preference set to WebGl; check that quads, shadows, paths, underlines, text (mono sprites), images (poly sprites) draw right side up and match the WebGPU build, and that resizing the window redraws at the new size.
- YOU: in the same WebGL run, force a context loss (`WEBGL_lose_context` extension's `loseContext()` from the devtools console) and check the console logs "wgpu device lost" once.
