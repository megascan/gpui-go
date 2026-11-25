# Parity: macros

Scope: gpui_macros. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_macros/src/gpui_macros.rs

Go target: `internal/cmd/(index)` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Action` | proc-macro-derive | 20 |  | `see DESIGN §9 (Action)` | dev:D-MACRO |
| `register_action` | proc-macro | 28 |  | `see DESIGN §9 (register_action)` | dev:D-MACRO |
| `IntoElement` | proc-macro-derive | 35 |  | `see DESIGN §9 (IntoElement)` | dev:D-MACRO |
| `Render` | proc-macro-derive | 41 |  | `see DESIGN §9 (Render)` | dev:D-MACRO |
| `AppContext` | proc-macro-derive | 59 |  | `see DESIGN §9 (AppContext)` | dev:D-MACRO |
| `VisualContext` | proc-macro-derive | 92 |  | `see DESIGN §9 (VisualContext)` | dev:D-MACRO |
| `style_helpers` | proc-macro | 99 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `visibility_style_methods` | proc-macro | 105 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `margin_style_methods` | proc-macro | 111 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `padding_style_methods` | proc-macro | 117 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `position_style_methods` | proc-macro | 123 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `overflow_style_methods` | proc-macro | 129 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `cursor_style_methods` | proc-macro | 135 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `border_style_methods` | proc-macro | 141 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `box_shadow_style_methods` | proc-macro | 147 |  | `internal/cmd/stylegen` -> `styled_gen.go` (`Styled[S]` methods, see DESIGN §9) | dev:D-MACRO |
| `test` | proc-macro-attribute | 189 |  | `see DESIGN §9 (test)` | dev:D-MACRO |
| `bench` | proc-macro-attribute | 222 |  | `RunBench`, `RunBenchRng`, `RunBenchInputs`, `RunBenchInputsRng` (see DESIGN §9 (bench)) | dev:D-BENCH |
| `property_test` | proc-macro-attribute | 295 |  | `see DESIGN §9 (property_test)` | dev:D-MACRO |
| `derive_inspector_reflection` | proc-macro-attribute | 317 | feat:inspector,debug | `InspectorReflectionMethods`, `InspectorReflectionFindMethod`, `StyledReflectionMethods`, `StyledReflectionFindMethod` (reflect over the method set, see DESIGN §9) | dev:D-MACRO |

## gpui_macros/tests/derive_context.rs

Go target: `macros_derive_context_test.go` (package `gpui_test`), tests in `macros_derive_context_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_context` | test | 2 |  | `TestDeriveContext_DeriveContext` | done |

## gpui_macros/tests/derive_inspector_reflection.rs

Go target: `macros_derive_inspector_reflection_test.go` (package `gpui_test`), tests in `macros_derive_inspector_reflection_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_inspector_reflection` | test | 68 |  | `TestDeriveInspectorReflection_DeriveInspectorReflection` | done |

## gpui_macros/tests/render_test.rs

Go target: `macros_render_test_test.go` (package `gpui_test`), tests in `macros_render_test_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_derive_render` | test | 2 |  | `TestRenderTest_DeriveRender` | done |


## Notes (M3-root-capture)

- `derive_inspector_reflection`: `InspectorReflectionMethods[T](trait reflect.Type, docs map[string]string)` and `InspectorReflectionFindMethod` reflect at run time over T's methods with the shape `func(T) T`, limited to the names in `trait`'s method set (upstream's trait). Go has no run-time doc comments, so documentation comes from the `docs` map. `StyledReflectionMethods[S]` / `StyledReflectionFindMethod[S]` are `styled_reflection::methods` / `find_method`: the trait is `*Styled[S]`, and the docs are the godoc of styled.go and styled_gen.go, embedded and parsed once. Names are Go method names (`Double`, not `double`); Go's reflect lists methods in sorted order, not declaration order.
