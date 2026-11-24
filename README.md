# gpui-go

A Go port of [GPUI](https://github.com/zed-industries/zed/tree/main/crates/gpui), the GPU-accelerated UI framework behind the [Zed](https://zed.dev) editor.

[![Go Reference](https://pkg.go.dev/badge/github.com/megascan/gpui-go.svg)](https://pkg.go.dev/github.com/megascan/gpui-go)

gpui-go tracks upstream GPUI at Zed commit `6501df15`. Types and functions keep their upstream names, converted to Go casing, and behave the same way. If you know GPUI in Rust, [Coming from Rust GPUI](docs/guide/from-rust-gpui.md) maps one onto the other.

## Status

| Platform | State |
|---|---|
| Linux (Wayland, X11) | Works. The examples run and are checked by screenshot. |
| macOS, Windows, iOS, web (`GOOS=js`) | Ported and compiling, but not yet run on real hardware or in a browser. |

The macOS, Windows, iOS and web backends are checked by cross-compiling, `go vet` and building the tests. Windows unit tests also run under Wine. Runs on real machines are pending; see [TECH_DEBT.md](TECH_DEBT.md#pending-real-os-tests). Known limits are listed in [Platforms](docs/guide/platforms.md#known-limits).

There is no tagged release yet, and the API can still change.

## Install

You need Go 1.27.3 or newer.

There are no version tags yet. Track the `dev` branch:

```sh
go get github.com/megascan/gpui-go@dev
```

Or work against a local checkout with a `replace` directive in your `go.mod`:

```
require github.com/megascan/gpui-go v0.0.0

replace github.com/megascan/gpui-go => ../gpui-go
```

Then run `go mod tidy` to fill in `go.sum`.

### System requirements

**Linux**

- cgo and a C compiler (`gcc` or `clang`). The renderer links wgpu-native through cgo, so `CGO_ENABLED=0` does not build.
- At run time: `libwayland-client.so.0`, `libwayland-cursor.so.0` and `libxkbcommon.so.0` for Wayland; `libxkbcommon-x11.so.0` and `libxcb.so.1` for X11; a Vulkan driver (OpenGL works as a fallback). These are loaded with `dlopen`, so no development headers are needed.

**macOS**: no cgo needed. Uses Metal.

**Windows**: no cgo needed. Uses Direct3D 11, DirectComposition and DirectWrite.

**Web**: build with `GOOS=js GOARCH=wasm`. Needs a browser with WebGPU or WebGL2.

Details for each platform are in [Platforms](docs/guide/platforms.md).

## Hello, world

From [examples/hello_world/main.go](examples/hello_world/main.go):

```go
package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

// HelloWorld is the hello_world example view.
type HelloWorld struct {
	text gpui.SharedString
}

func swatch(bg, border gpui.Hsla) *gpui.Div {
	return gpui.NewDiv().
		Size8().
		Bg(bg).
		Border1().
		BorderDashed().
		RoundedMd().
		BorderColor(border)
}

// Render renders the view.
func (h *HelloWorld) Render(_ *gpui.Window, _ *gpui.Context[HelloWorld]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		Border1().
		BorderColor(gpui.Rgb(0x0000ff)).
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText(fmt.Sprintf("Hello, %s!", h.text)).
		Child(
			gpui.NewDiv().
				Flex().
				Gap2().
				Child(swatch(gpui.Red(), gpui.White())).
				Child(swatch(gpui.Green(), gpui.White())).
				Child(swatch(gpui.Blue(), gpui.White())).
				Child(swatch(gpui.Yellow(), gpui.White())).
				Child(swatch(gpui.Black(), gpui.White()).RoundedMd()).
				Child(swatch(gpui.White(), gpui.Black())),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[HelloWorld] {
			return cx.New(func(_ *gpui.Context[HelloWorld]) *HelloWorld {
				return &HelloWorld{text: "World"}
			})
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
```

Run it from a checkout:

```sh
go run ./examples/hello_world
```

## Learn more

- [Getting started](docs/guide/getting-started.md): build a counter app step by step.
- [The guide](docs/guide/README.md): styling, actions and focus, async, lists, testing, platforms, concepts and pitfalls.
- [Examples](examples/): about 50 runnable programs, from layout and text to drag and drop and window management.
- [API reference](https://pkg.go.dev/github.com/megascan/gpui-go) on pkg.go.dev.

## Credits

gpui-go is a port of GPUI by Zed Industries, Inc., which is licensed under the Apache License 2.0. It also ports parts of several Rust crates that GPUI depends on, such as taffy, lyon and accesskit. [NOTICE](NOTICE) lists them with their licenses.

## License

Apache License 2.0. See [LICENSE](LICENSE).
