# Platforms

`platform.Application()` picks the backend for the OS you build for. This page lists what each backend needs, how to choose between backends where there is a choice, and what is known not to work.

| OS | Windowing | Renderer | cgo | Status |
|---|---|---|---|---|
| Linux | Wayland or X11 | wgpu (Vulkan, or GL) | required | runs; the examples are checked by screenshot |
| macOS | AppKit | Metal | no | compiles and passes `go vet` and `go test -c`; not run on a Mac yet |
| Windows | Win32 | Direct3D 11 + DirectComposition | no | compiles; unit tests run under Wine; not run on Windows yet |
| Web | browser canvas | WebGPU, WebGL2 fallback | no | compiles; not checked in a browser yet |
| iOS | UIKit | Metal | required | compiles; not run on a simulator or device yet |

The unverified rows are tracked in [TECH_DEBT.md](../../TECH_DEBT.md#pending-real-os-tests). If you run gpui-go on one of these platforms, reports are welcome.

On any other `GOOS`, `platform.Application()` panics with "gpui: the current platform is not ported yet".

## Linux

### Build requirements

- Go 1.27.3 or newer.
- cgo and a C compiler. The renderer links wgpu-native statically through cgo, so `CGO_ENABLED=0` does not build.

### Runtime libraries

These are loaded at startup with `dlopen`, so they are needed on the machine that runs the program, not at build time:

| Library | Needed for |
|---|---|
| `libwayland-client.so.0`, `libwayland-cursor.so.0` | Wayland |
| `libxkbcommon.so.0` | keyboard handling |
| `libxkbcommon-x11.so.0`, `libxcb.so.1` | X11 |
| `libpipewire-0.3.so.0` | screen capture only |
| a Vulkan driver (or OpenGL) | rendering |

### Wayland or X11

At startup the Linux platform picks, in order:

1. Wayland, if `WAYLAND_DISPLAY` is set and not empty.
2. X11, if `DISPLAY` is set and not empty.
3. Headless (no windows), otherwise.

Setting `ZED_HEADLESS` to any value forces headless mode.

To force X11 on a Wayland session for one run, clear `WAYLAND_DISPLAY`:

```sh
WAYLAND_DISPLAY= go run ./examples/hello_world
```

To restrict the choice in code, use `platform.Linux` with the modes you allow, for example `platform.Linux(gpui.WindowingModesX11 | gpui.WindowingModesHeadless)`. `App.RequestWindowing` switches mode at runtime; [examples/toggle_headless](../../examples/toggle_headless/main.go) shows it.

Other environment variables:

- `ZED_DEVICE_ID` picks a GPU by device id when you have more than one.
- `GPUI_X11_SCALE_FACTOR` overrides the X11 scale factor.
- `XCURSOR_THEME`, `XCURSOR_SIZE` and `XCURSOR_PATH` set the cursor theme.

Wayland layer-shell surfaces (panels, overlays) are supported; see [examples/layer_shell](../../examples/layer_shell/main.go).

### Renderer

Linux uses wgpu-native v29 through [oliverbestmann/webgpu](https://github.com/oliverbestmann/webgpu). Zed's GPUI targets wgpu 30. Where v29 lacks a feature (adapter enumeration, dual-source blending, transient attachments), gpui-go uses the fallback paths (limit L-WGPUVER). The renderer logs the adapter it picked at startup.

## macOS

- No cgo. The AppKit and Metal bindings go through [purego](https://github.com/ebitengine/purego), so `CGO_ENABLED=0` builds work, including cross-builds from Linux.
- Metal shaders are compiled from source when the app starts.
- Native views (`examples/native_view`, `examples/native_view_appkit_field`) are macOS only.
- Application menus from `cx.SetMenus` show in the macOS menu bar.

Status: not yet run on a Mac. The code is checked by cross-compiling, `go vet` and `go test -c` only.

## Windows

- No cgo. Win32, COM, Direct3D 11, DirectComposition and DirectWrite are called through `golang.org/x/sys/windows`.
- HLSL shaders are compiled when the app starts, with `D3DCompile` from `d3dcompiler_47.dll`.
- Set `GPUI_DISABLE_DIRECT_COMPOSITION=1` (or `true`) to render without DirectComposition.
- There is no headless renderer on Windows yet: `platform.CurrentHeadlessRenderer` returns nil.

Status: not yet run on Windows. Unit tests run under Wine, where a few tests fail because Wine lacks some DirectWrite, UI Automation and WinRT functions (see [TECH_DEBT.md](../../TECH_DEBT.md#wine-test)).

## Web (GOOS=js)

Build to WebAssembly:

```sh
GOOS=js GOARCH=wasm go build -o web/app.wasm ./examples/web_hello_web
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/
```

Serve the directory with an HTML page that loads `wasm_exec.js` and runs the module. [examples/web_hello_web](../../examples/web_hello_web/) has an `index.html` and `gallery.js` you can copy.

Three things differ from desktop:

- `Run` returns instead of blocking, so `main` must not exit. End it with `select {}`, as [examples/web_hello_web/main.go](../../examples/web_hello_web/main.go) does:

```go
func main() {
	platform.WebInit()
	name := ""
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	switch name {
	case "dynamic-fonts":
		RunDynamicFonts()
	default:
		runHelloWeb()
	}
	select {}
}
```

- The browser has no system fonts for GPUI to use. Load font files before you render text:

```go
//go:embed fonts/IBMPlexSans-Regular.ttf
var IBMPlexSansRegular []byte

func runHelloWeb() {
	runApplication(func(cx *gpui.App) {
		if err := cx.TextSystem().AddFonts([][]byte{IBMPlexSansRegular}); err != nil {
			js.Global().Get("console").Call("error", fmt.Sprintf("failed to load application fonts: %v", err))
			return
		}
	})
}
```

- There is one thread (limit L-WEBTHREADS). Background tasks run on the same thread as the UI, so long CPU work still blocks rendering. Split it into steps that yield.

The renderer uses WebGPU when the browser has it and falls back to WebGL2. `platform.ApplicationWithWebBackend` forces one or the other.

Status: compiles (`go vet` and `go test -c` with `GOOS=js`). Not yet checked in a browser.

## iOS

The iOS backend (UIKit + Metal) is ported but not yet run. Building needs macOS with Xcode, because the Go code is built as a C archive and linked into an app bundle. [examples/ios_ios/build-simulator.sh](../../examples/ios_ios/build-simulator.sh) builds the example for the simulator.

## Known limits

These differences from Zed's GPUI are accepted and will not be fixed in this port (full list in [PARITY.md section 5](../port/PARITY.md#5-deviations-and-limits)):

| Id | Limit |
|---|---|
| L-GLYPH | Linux glyph rasterization differs slightly in pixels from Zed's (different font rasterizer). |
| L-JPEG | JPEG decoding can produce slightly different pixel values. |
| L-SVGTEXT | Text and embedded images inside SVG files are not drawn. |
| L-WGPUVER | wgpu-native v29 instead of wgpu 30, with fallbacks for missing features. |
| L-WEBTHREADS | No multi-threaded web dispatcher. |

Other known issues are listed in [TECH_DEBT.md](../../TECH_DEBT.md). One you may hit: on Hyprland, `examples/window_positioning` sometimes aborts inside `wgpuSurfaceConfigure`.
