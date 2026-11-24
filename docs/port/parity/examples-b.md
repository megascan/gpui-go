# Parity: examples-b

Scope: gpui: examples (text_layout through painting). Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

No rows: the scan found no public items or tests in these upstream files (PARITY.md section 4). Owners add a row for any item or test the scan missed (PARITY.md section 8, step 5).

## Notes (M3-examples-b)

- Each example is one `package main` under `examples/<name>/main.go`; none of the upstream files has tests. `actions!` types are registered by hand from `init`, as `examples/image_example` does.
- painting: `lyon_extra::rust_logo::build_logo_path` (lyon_extra 1.1.0 from crates.io, not in the pinned registry list) is ported into `examples/painting/rust_logo.go` against `internal/lyon.SvgPathBuilder`. Upstream converts the lyon builder with `PathBuilder::from(WithSvg<BuilderImpl>)`; the Go conversion is unexported, so the example replays the built lyon path's events into `gpui.PathBuilderFill()` (D-EXB-SVGINTO; handoff M3-examples-b).
- mouse_pressure: `{:?}` of `PressureStage` is a local helper printing the variant name, since `gpui.PressureStage` has no `String` (handoff M3-examples-b).
- a11y: `env_logger` (Warn, `gpui` at Info) becomes `slog.SetLogLoggerLevel(slog.LevelInfo)`; slog has no per-module filter.
- Examples gated only by `target_family = "wasm"` upstream (the `start` entry points) are covered by the web example stage; these packages build for js/wasm through the same `main`.

## Deviations registered here

| Id | Deviation | Upstream | Reason |
|---|---|---|---|
| D-EXB-SVGINTO | painting builds the Rust logo with `lyon.PathSvgBuilder()` and replays its events into `gpui.PathBuilderFill()` instead of converting the lyon builder | `gpui/examples/painting.rs:184-190` | `From<WithSvg<BuilderImpl>> for PathBuilder` is unexported in Go (`newPathBuilderFromSvg`) |
