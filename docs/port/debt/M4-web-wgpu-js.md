# M4-web-wgpu-js debt

- WebGL2 branch of `NewWgpuContextWeb` returns `errWebGlUnavailable` until M4-web-webgl (D-WEBGLPENDING).
- oliverbestmann/webgpu js `Adapter.GetInfo` is empty: adapter name, vendor and device are blank in logs and `CheckCompatibleWithSurface` messages on the web.
- If the awaiting task unwinds before the promises resolve, the helper goroutine releases the graphics it created; promises themselves cannot be aborted.
