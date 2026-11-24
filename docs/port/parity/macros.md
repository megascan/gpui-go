# Parity: macros

Scope: gpui_macros. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_macros/src/gpui_macros.rs

Go target: `internal/cmd/(index)` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Action` | proc-macro-derive | 20 |  | `see DESIGN §9 (Action)` | dev:D-MACRO |
| `register_action` | proc-macro | 28 |  | `see DESIGN §9 (register_action)` | dev:D-MACRO |
| `IntoElement` | proc-macro-derive | 35 |  | `see DESIGN §9 (IntoElement)` | todo |
| `Render` | proc-macro-derive | 41 |  | `see DESIGN §9 (Render)` | todo |
| `AppContext` | proc-macro-derive | 59 |  | `see DESIGN §9 (AppContext)` | dev:D-MACRO |
| `VisualContext` | proc-macro-derive | 92 |  | `see DESIGN §9 (VisualContext)` | dev:D-MACRO |
| `style_helpers` | proc-macro | 99 |  | `see DESIGN §9 (style_helpers)` | todo |
| `visibility_style_methods` | proc-macro | 105 |  | `see DESIGN §9 (visibility_style_methods)` | todo |
| `margin_style_methods` | proc-macro | 111 |  | `see DESIGN §9 (margin_style_methods)` | todo |
| `padding_style_methods` | proc-macro | 117 |  | `see DESIGN §9 (padding_style_methods)` | todo |
| `position_style_methods` | proc-macro | 123 |  | `see DESIGN §9 (position_style_methods)` | todo |
| `overflow_style_methods` | proc-macro | 129 |  | `see DESIGN §9 (overflow_style_methods)` | todo |
| `cursor_style_methods` | proc-macro | 135 |  | `see DESIGN §9 (cursor_style_methods)` | todo |
| `border_style_methods` | proc-macro | 141 |  | `see DESIGN §9 (border_style_methods)` | todo |
| `box_shadow_style_methods` | proc-macro | 147 |  | `see DESIGN §9 (box_shadow_style_methods)` | todo |
| `test` | proc-macro-attribute | 189 |  | `see DESIGN §9 (test)` | dev:D-MACRO |
| `bench` | proc-macro-attribute | 222 |  | `see DESIGN §9 (bench)` | todo |
| `property_test` | proc-macro-attribute | 295 |  | `see DESIGN §9 (property_test)` | dev:D-MACRO |
| `derive_inspector_reflection` | proc-macro-attribute | 317 | feat:inspector,debug | `see DESIGN §9 (derive_inspector_reflection)` | todo |

## gpui_macros/tests/derive_context.rs

Go target: `macros_derive_context_test.go` (package `gpui_test`), tests in `macros_derive_context_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_context` | test | 2 |  | `TestDeriveContext_DeriveContext` | todo |

## gpui_macros/tests/derive_inspector_reflection.rs

Go target: `macros_derive_inspector_reflection_test.go` (package `gpui_test`), tests in `macros_derive_inspector_reflection_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_inspector_reflection` | test | 68 |  | `TestDeriveInspectorReflection_DeriveInspectorReflection` | todo |

## gpui_macros/tests/render_test.rs

Go target: `macros_render_test_test.go` (package `gpui_test`), tests in `macros_render_test_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_render` | test | 2 |  | `TestRenderTest_DeriveRender` | todo |

