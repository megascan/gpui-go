# Platforms, builds and environment

## Contents
- Status
- Toolchain and module
- Linux
- macOS
- Windows
- Web (wasm)
- Build tags and env vars
- Known limits

## Status

- Verified on Linux only. Every Linux example builds, runs and has a checked screenshot.
- macOS, iOS and Windows are checked only by cross `go vet` and `go test -c` (plus Wine for some Windows tests). No real-OS run has happened yet. Treat behaviour there as unverified and say so when it matters, for example `Show` handling, menus or native views.
- Any other GOOS returns a "platform not ported" error.

## Toolchain and module

- Go 1.27.3 or newer, because the port relies on generic methods on concrete types.
- No version tags are published. Use `require github.com/megascan/gpui-go v0.0.0` plus a `replace` pointing at a checkout. Copy the checkout's `go.sum` and `require` blocks (including the indirect ones such as `oliverbestmann/webgpu/libs-linux`), or run `go mod tidy` in your own module.
- `examples/internal/support` and everything under `internal/` can't be imported from another module. The exception is `go run github.com/megascan/gpui-go/internal/cmd/actiongen`, which works from an external module.
- Build checks for other OSes, run from Linux:
  `CGO_ENABLED=0 GOOS=darwin go vet ./...`, `CGO_ENABLED=0 GOOS=windows go vet ./...`, `CGO_ENABLED=0 GOOS=js GOARCH=wasm go vet ./...`.

## Linux

- **cgo is required.** `render/wgpu` links wgpu-native statically through `github.com/oliverbestmann/webgpu`. With `CGO_ENABLED=0` the build fails with `build constraints exclude all Go files in .../wgpu/internal/needscgo`. Install a C toolchain.
- Runtime libraries are loaded with dlopen: `libwayland-client.so.0`, `libwayland-cursor.so.0`, `libxkbcommon.so.0`, `libxkbcommon-x11.so.0`, `libxcb.so.1` (X11), and `libpipewire-0.3.so.0` (screen capture only). A Vulkan driver is needed, with GLES as the fallback.
- Backend choice: Wayland if `WAYLAND_DISPLAY` is set, else X11 if `DISPLAY` is set, else headless. `ZED_HEADLESS` (any value) forces headless. To force a backend from code:

```go
func main() {
	app := platform.Linux(gpui.WindowingModesX11 | gpui.WindowingModesHeadless)
	app.Run(func(cx *gpui.App) {
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Root] {
			return cx.New(func(_ *gpui.Context[Root]) *Root { return &Root{} })
		}); err != nil {
			panic(err)
		}
	})
}
```

  Put it in a `//go:build linux` file. Switch at runtime with `cx.RequestWindowing(...)`.
- Wayland layer-shell surfaces (bars, overlays) work only on Wayland. See examples/layer_shell.
- `WindowOptions.Icon` (`Option[*image.NRGBA]`) works on X11 only.
- Known issue: on Hyprland, window_positioning sometimes aborts inside `wgpuSurfaceConfigure`.

## macOS

- No cgo: purego and objc for AppKit, Metal for rendering, shaders compiled at startup.
- Windows only appear when `WindowOptions.Show` is true. This is the main reason to use `gpui.DefaultWindowOptions()`.
- Menus (`cx.SetMenus`) are visible only on macOS. Native views (examples/native_view*) are macOS-only.

## Windows

- No cgo: x/sys/windows plus hand-written COM, with D3D11, DirectComposition and DirectWrite. HLSL is compiled at startup with `d3dcompiler_47`.
- There is no headless renderer, so screenshots go through a real window.
- `GPUI_DISABLE_DIRECT_COMPOSITION` disables DirectComposition.

## Web (wasm)

- Build with `GOOS=js GOARCH=wasm go build -o app.wasm .` and serve the binary next to `$(go env GOROOT)/lib/wasm/wasm_exec.js`.
- `Run` returns on the web, so `main` must block with `select {}`.
- There are no system fonts. Embed a font and call `cx.TextSystem().AddFonts(...)` before you render text.
- Rendering uses WebGPU and falls back to WebGL2. It is single-threaded only (L-WEBTHREADS). Choose the backend with `platform.ApplicationWithWebBackend(pref)`.

```go
//go:embed font.ttf
var font []byte

type Root struct{}

func (*Root) Render(_ *gpui.Window, _ *gpui.Context[Root]) gpui.IntoElement {
	return gpui.NewDiv().SizeFull().ChildText("hello from wasm")
}

func main() {
	platform.WebInit()
	platform.Application().Run(func(cx *gpui.App) {
		if err := cx.TextSystem().AddFonts([][]byte{font}); err != nil {
			panic(err)
		}
		if _, err := cx.OpenWindow(gpui.DefaultWindowOptions(), func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Root] {
			return cx.New(func(_ *gpui.Context[Root]) *Root { return &Root{} })
		}); err != nil {
			panic(err)
		}
	})
	select {}
}
```

(This needs a `//go:build js && wasm` file plus an `import _ "embed"`.)

## Build tags and env vars

| Name | Effect |
|---|---|
| `-tags gpui_release` | Turns debug assertions off. `DebugSelector` becomes a no-op and double-`Hover` no longer panics |
| `-tags gpui_profiler` | Enables the profiler (replaces the cargo feature) |
| `ZED_HEADLESS` | Forces the headless platform on Linux |
| `WAYLAND_DISPLAY`, `DISPLAY` | Backend selection |
| `GPUI_X11_SCALE_FACTOR` | X11 scale override |
| `ZED_DEVICE_ID` | Picks the wgpu GPU |
| `GPUI_LEAK_DETECTION` | `strict`, or any non-empty value for "on". Unset inside tests means strict |
| `LEAK_BACKTRACE` | Records allocation stacks for leak reports |
| `SEED`, `ITERATIONS`, `PROPTEST_CASES` | Seeded and randomized tests |
| `GPUI_TEST_DISPLAY=1` | Gates gpui-go's own display tests. The library doesn't read it for your tests |

## Known limits

| Id | Limit |
|---|---|
| L-GLYPH | Linux glyph pixels differ slightly from upstream's swash rasterizer |
| L-JPEG | JPEG decode pixel values may differ |
| L-SVGTEXT | Text and embedded images inside SVGs are not rendered |
| L-WGPUVER | wgpu-native v29 vs upstream wgpu 30. Some features fall back |
| L-WEBTHREADS | No multi-threaded web dispatcher |
