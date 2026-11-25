# gpui-go

gpui-go is a Go port of [GPUI](https://github.com/zed-industries/zed/tree/main/crates/gpui), the GPU-accelerated UI framework that the [Zed](https://zed.dev) editor is built on.

[![Go Reference](https://pkg.go.dev/badge/github.com/megascan/gpui-go.svg)](https://pkg.go.dev/github.com/megascan/gpui-go)

The port follows upstream GPUI at Zed commit [`6501df15`](https://github.com/zed-industries/zed/tree/6501df15765d66d6c2a2b66b1eb22434ccef3f86/crates/gpui). Types and functions keep their upstream names in Go casing and behave the same way. Where Go forces a different shape, the change is a numbered deviation (see [Parity with upstream](#parity-with-upstream)). If you know GPUI in Rust, start with [Coming from Rust GPUI](docs/guide/from-rust-gpui.md).

## Status

| Platform | State |
|---|---|
| Linux (Wayland, X11) | Tested. Every example runs, and each is checked with a screenshot. |
| Windows | Tested on Windows 11. The full test suite passes, and the examples run. IME composition and toast banners are unconfirmed. |
| macOS | Ported and cross-compiled. Not yet run on real hardware. |
| iOS, web (`GOOS=js`) | Ported and cross-compiled. Not yet run on a device or in a browser. |

macOS, iOS and web are checked by cross-compiling, `go vet` and building their tests. Expect bugs there until they have been run on real machines; the open items are tracked in [TECH_DEBT.md](TECH_DEBT.md#pending-real-os-tests), and known limits are in [Platforms](docs/guide/platforms.md#known-limits).

There is no tagged release yet, and the API may still change.

## Install

Requires Go 1.27.3 or newer.

There are no version tags yet, so track the `dev` branch:

```sh
go get github.com/megascan/gpui-go@dev
```

To work against a local checkout instead, add a `replace` directive to your `go.mod` and run `go mod tidy`:

```
require github.com/megascan/gpui-go v0.0.0

replace github.com/megascan/gpui-go => ../gpui-go
```

### System requirements

- **Linux**: cgo and a C compiler (`gcc` or `clang`), because the renderer links wgpu-native through cgo; `CGO_ENABLED=0` does not build. At run time it loads `libwayland-client.so.0`, `libwayland-cursor.so.0` and `libxkbcommon.so.0` (Wayland) or `libxkbcommon-x11.so.0` and `libxcb.so.1` (X11), plus a Vulkan driver, with OpenGL as a fallback. These are loaded with `dlopen`, so no development headers are needed.
- **macOS**: no cgo. Renders with Metal.
- **Windows**: no cgo. Uses Direct3D 11, DirectComposition and DirectWrite.
- **Web**: build with `GOOS=js GOARCH=wasm`. Needs a browser with WebGPU or WebGL2.

Per-platform details are in [Platforms](docs/guide/platforms.md).

## Hello, world

```go
package main

import (
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

type Hello struct{}

func (h *Hello) Render(_ *gpui.Window, _ *gpui.Context[Hello]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		Size(gpui.Px(400)).
		JustifyCenter().
		ItemsCenter().
		Bg(gpui.Rgb(0x1e1e1e)).
		TextColor(gpui.Rgb(0xffffff)).
		TextXl().
		ChildText("Hello, world!")
}

func main() {
	platform.Application().Run(func(cx *gpui.App) {
		options := gpui.DefaultWindowOptions()
		_, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[Hello] {
			return cx.New(func(_ *gpui.Context[Hello]) *Hello { return &Hello{} })
		})
		if err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}
```

`platform.Application()` picks the backend for the current OS. The window's root view is an entity, and its `Render` method returns a fresh element tree each frame.

## Examples

[examples/](examples/) has about 50 programs ported from upstream, covering layout, text, images, SVG, lists, input, drag and drop, animation and window management. Run one from a checkout:

```sh
go run ./examples/hello_world
go run ./examples/counter
go run ./examples/uniform_list
```

Some examples only build on one platform (for example `layer_shell` on Wayland, `native_view_appkit_field` on macOS, `web_hello_web` for `GOOS=js`).

## Documentation

- [Getting started](docs/guide/getting-started.md): build a counter app step by step.
- [The guide](docs/guide/README.md): styling, actions and focus, async work, lists, testing, platforms, concepts and common pitfalls.
- [API reference](https://pkg.go.dev/github.com/megascan/gpui-go) on pkg.go.dev.

The code in the guide is compiled and tested as part of the repo.

## Parity with upstream

The goal is functional parity with GPUI at `zed@6501df15765d66d6c2a2b66b1eb22434ccef3f86`. [docs/port/PARITY.md](docs/port/PARITY.md) maps every upstream public item and test to a Go symbol, a Go test or a registered deviation, split by area under [docs/port/parity/](docs/port/parity/).

Where Go cannot express the Rust API directly, the port uses deviations with `D-*` ids, for example `D-RAII` (explicit `Release` instead of `Drop`), `D-OPT` (Option and Result) and `D-ASYNC` (coroutines in place of futures). They are defined in [DESIGN.md section 5](docs/port/DESIGN.md) and listed in [PARITY.md section 5](docs/port/PARITY.md#5-deviations-and-limits), together with the known limits (`L-*`).

## Security

The library has been through two rounds of security review. The threat model, findings and fixes are in [docs/security/LEDGER.md](docs/security/LEDGER.md). Fixes that depart from upstream behaviour are registered as deviations.

## Claude Code skill

[skills/gpui-go/](skills/gpui-go/) is a [Claude Code](https://docs.anthropic.com/en/docs/claude-code) skill for writing, testing and debugging gpui-go apps. To install it, copy it into your skills directory:

```sh
mkdir -p ~/.claude/skills
cp -r skills/gpui-go ~/.claude/skills/
```

To use it in one project only, copy it to that project's `.claude/skills/` instead.

## Credits and license

gpui-go is a port of GPUI by Zed Industries, Inc., licensed under the Apache License 2.0. It also ports parts of Rust crates that GPUI depends on, including taffy, lyon, cosmic-text and accesskit. [NOTICE](NOTICE) lists each one with its license and copyright.

gpui-go is licensed under the Apache License 2.0. See [LICENSE](LICENSE).
