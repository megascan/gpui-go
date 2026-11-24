# gpui-go guide

Start with the tutorial, then use the guides when you need them.

## Tutorial

- [Getting started](getting-started.md): build a counter app, step by step.

## How-to guides

- [Elements and styling](elements-and-styling.md): layout, lengths, colors, hover styles, components, images, animation.
- [Actions, keymaps and focus](actions-keymaps-focus.md): keyboard input, key bindings, focus and tab order.
- [Async work and tasks](async-and-tasks.md): timers, background work, cancellation.
- [Lists and scrolling](lists-and-scrolling.md): scrollable divs, uniform lists, variable-height lists.
- [Testing](testing.md): test apps, simulated input, the fake clock, seeds and leak detection.
- [Platforms](platforms.md): Linux (Wayland and X11), macOS, Windows, web and iOS; requirements and known limits.

## Explanation

- [Concepts](concepts.md): entities, contexts, the effect queue, the render cycle, `Release` and `Drop`.
- [Coming from Rust GPUI](from-rust-gpui.md): how the Rust API maps to Go, and the deviations.
- [Pitfalls](pitfalls.md): common mistakes and their fixes.

## Reference

- API documentation: [pkg.go.dev/github.com/megascan/gpui-go](https://pkg.go.dev/github.com/megascan/gpui-go).
- [Examples](../../examples/): about 50 runnable programs. Run one with `go run ./examples/<name>`.
- Background docs ported from Zed: [contexts](../contexts.md), [key dispatch](../key-dispatch.md), [ownership and data flow](../ownership-and-data-flow.md), [accessibility](../accessibility.md).

The code in these guides is compiled and tested as part of the repo, under [snippets/](snippets/) and [examples/counter](../../examples/counter/main.go).
