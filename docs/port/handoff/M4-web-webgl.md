# Handoff: M4-web-webgl

- STAGES.md says wgpu rev cddb2f5 "adds the WebGL canvas-resize fix wgpu#10348". In the cargo git checkout, cddb2f5 is "Ignore timeouts on wasm (#10348)" and only touches wgpu-core/src/instance.rs. The webgl2 facade follows that: `GetCurrentTexture` never returns Timeout. Its canvas resize follows wgpu-hal's web.rs `configure` at that rev, which sets the canvas width and height. An exit stage should correct the STAGES.md wording.
- `go_js_wasm_exec` fails with "total length of command line and environment variables exceeds limit" when the environment is larger than Go's wasm limit. This shell's environment is about 9 KB. The verify for this stage was run with `env -i HOME PATH GOPATH GOCACHE`. verify.sh and CI may need the same trimming, which is for the exit or CI stages to decide.

## verify.sh: wasm test fails on large environments

go_js_wasm_exec copies argv+env into wasm memory below wasmMinDataAddr (12288 bytes, minus argv). The builder shell's environment is about 9 KB, so every js/wasm test binary aborts with "total length of command line and environment variables exceeds limit" before any test runs. The webgl2 tests pass with a trimmed environment:

    env -i PATH="$PATH" HOME="$HOME" GOCACHE="$(go env GOCACHE)" GOPATH="$(go env GOPATH)" GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" ./render/wgpu/internal/gpu/webgl2/...

Fix for an exit stage (verify.sh lines 486 and 527, with STAGES.md): run the wasm test step under `env -i` with PATH, HOME, GOCACHE, GOPATH, GOMODCACHE, GOFLAGS.
