# Dependency research (2026-11-23)

Upstream target: zed-industries/zed @ 6501df15765d66d6c2a2b66b1eb22434ccef3f86. Reference checkout: `_ref/` (symlink, gitignored).

| Rust dep | Go choice | Identical? |
|---|---|---|
| taffy =0.15.0 | Port 1:1 (~15-20k Go LOC; flex 3.5k, grid 8.2k, block 1.8k, style 7k, tree 3.4k) + port its generated fixture tests. No Go taffy exists. | Yes |
| cosmic-text 0.19 + swash (Linux/wgpu text, gpui_wgpu/src/cosmic_text_system.rs) | go-text/typesetting v0.3.5 (shaping, bidi, UAX14/29, fontscan fallback). Raster via xo/okage tiny-skia port or x/image/vector | Shaping ~same; glyph pixels differ from swash (swash scaler ~26.5k LOC) |
| CoreText (mac, gpui_apple) / DirectWrite (win) | Same OS APIs via purego/cgo | Yes |
| resvg/usvg 0.46 | xo/okage v0.20.1 (pure-Go tiny-skia + resvg port) | Yes except text/images |
| icu_segmenter 2.3, unicode-segmentation | go-text/typesetting/segmenter (rivo/uniseg stale) | Mostly; verify vs conformance tests |
| etagere 0.2 | Port (~2.7k LOC) | Yes |
| lyon_path / tiny-skia-path | Port what path_builder.rs uses; okage/path ports tiny-skia-path | Yes |
| ttf-parser | go-text/typesetting/font | Metrics close |
| image 0.25 | stdlib png/jpeg/gif, x/image/webp, gen2brain/avif (wasm/wazero, no cgo) | JPEG may differ slightly (Go 1.26 decoder) |
| accesskit 0.24 | cgo accesskit-c v0.23.2, or native AX/UIA/AT-SPI adapters | Check 0.23 vs 0.24 model |
| scheduler TestDispatcher (seeded StdRng, single thread) | Own scheduler: single-goroutine run queue, math/rand/v2 seeded PCG, virtual clock. testing/synctest (GA, 1.27 adds synctest.Sleep) gives virtual time but not seeded order. | Yes if we own the scheduler |

## Go language (local: go1.27.4)
- Generic methods on concrete types: Go 1.27 (not in interfaces; cannot satisfy interface methods).
- 1.27: struct literal keys may name nested field selectors; broader func type inference; Unicode 17; experimental simd.
- 1.26: `new(expr)`; self-referential generic constraints (`type Adder[A Adder[A]]`); Green Tea GC default; cgo calls ~30% cheaper.
- Stable: generic type aliases, range-over-func iterators, `unique`, `weak`, `runtime.AddCleanup`. Arenas: on hold, use per-frame slice pools.

## Known parity limits
Linux glyph pixels (unless swash scaler ported), JPEG decode, SVG text. Added by owner decision (DESIGN.md 14.3): L-WGPUVER (wgpu-native v29 vs upstream wgpu 30) and L-WEBTHREADS (no multi-threaded web dispatcher). The WebGL fallback is ported in-repo (WGSL -> GLSL ES via naga at generate time, a WebGL2 implementation of the renderer's GPU facade), so the binding's missing WebGL backend is not a limit.

# GPU / windowing research (2026-11-23)

Upstream renderers @6501df1: Linux = gpui_wgpu (wgpu 30 git cddb2f5, Vulkan+GLES; shaders.wgsl + shaders_storage.wgsl | shaders_webgl.wgsl; shaders_subpixel.wgsl needs dual-source blending). macOS = Metal direct (gpui_apple, shaders.metal, objc2). Windows = D3D11 + DirectComposition + DirectWrite (shaders.hlsl, alpha_correction.hlsl). Web = gpui_wgpu (WebGPU, WebGL fallback).

- GPU binding: github.com/oliverbestmann/webgpu v1.38.1 (2026-11-13; cogentcore's successor). cgo, static wgpu-native v29.0.1.1 (one major behind upstream v30). Vulkan/Metal/D3D12/GLES; GOOS=js uses browser WebGPU (no WebGL fallback). Alt: go-webgpu/webgpu (goffi, beta), gogpu/wgpu (pure-Go, young; track).
- Metal: purego/objc (purego v0.11.3). D3D11/DComp: pure-Go COM via x/sys/windows (Gio internal/d3d11 as reference, MIT/Unlicense).
- Wayland: own generator from protocol XML (neurlang/wayland needs C xkbcommon; go-wayland archived). xkbcommon via purego dlopen. X11: jezek/xgb v1.3.1 + xgbutil; XIM hand-written (like xim-rs).
- macOS windowing: purego/objc RegisterClass/AddMethod/AddProtocol (Ebitengine internal/cocoa model). Windows: x/sys/windows + hand COM vtables.
- cgo only for wgpu-native link -> CI matrix on native runners (ubuntu, macos arm64, windows). GPU layer behind an interface.
