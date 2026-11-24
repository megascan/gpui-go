# Parity: examples-c

Scope: gpui: examples (window through native_view_appkit_field). Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

No rows: the scan found no public items or tests in these upstream files (PARITY.md section 4). Owners add a row for any item or test the scan missed (PARITY.md section 8, step 5).

## Notes (M3-examples-c)

- Each example is one `package main` under `examples/<name>/`; none of the upstream files has tests. `actions!` types are registered by hand from `init`. Window options start from `gpui.DefaultWindowOptions()` (upstream's `..Default::default()`).
- OS gates: layer_shell's body is `example_linux.go`, other systems panic with upstream's message; native_view and native_view_appkit_field are `darwin && !ios` (`example_macos.go`) and print "Native views are only composited on macOS." elsewhere; toggle_headless keeps the per-OS `display_environment` / `USAGE` and has an empty main on js.
- img builder calls (`WithFallback`, `WithLoading`, `ObjectFit`) come before `.ID()`, since `Stateful[*Img]` does not expose StyledImage; they are plain setters, so the order does not matter.
- `env_logger::init()` has no Go counterpart; the default slog handler is used.

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-EXC-NVHANDLE | native_view and native_view_appkit_field pass the view's own `NativeView` handle as the child instead of `Clone()` | `gpui/examples/native_view.rs:464`, `native_view_appkit_field.rs:198` | the native view element never releases the handle it is built from, so a per-frame clone would grow the ref count (handoff M3-examples-c) |
| D-EXC-IVAR | the background reporter's sender lives in a Go map keyed by the reporter object instead of an objc ivar | `gpui/examples/native_view.rs:494-540` | purego class registration has no typed ivar access; the map gives the same per-object lookup |
