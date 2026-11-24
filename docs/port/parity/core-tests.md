# Parity: core-tests

Scope: gpui: integration tests and benches (gpui/build.rs, benches/focus_cleanup.rs, tests/action_macros.rs, tests/eq_hash.rs). Milestone: M0, M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/benches/focus_cleanup.rs

Go target: `bench_focus_cleanup_test.go` (package `gpui_test`), tests in `bench_focus_cleanup_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `no_drops` | gpui::bench(inputs = [0, 100, 1_000, 10_000], input_name = "handles", group = "focus_cleanup_no_drops_10000_effects", fps = 120) | 13 |  | `BenchmarkFocusCleanup_NoDrops` | done |
| `final_drops` | gpui::bench(inputs = [10_000], input_name = "handles", group = "focus_cleanup_final_drops_10000_effects", fps = 120) | 23 |  | `BenchmarkFocusCleanup_FinalDrops` | done |
| `final_drops_with_windows` | gpui::bench(inputs = [1, 10], input_name = "windows", group = "focus_cleanup_final_drops_with_windows_10000_handles_10000_effects", fps = 120) | 33 |  | `BenchmarkFocusCleanup_FinalDropsWithWindows` | done |

## gpui/tests/action_macros.rs

Go target: `integration_action_macros_test.go` (package `gpui_test`), tests in `integration_action_macros_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_action_macros` | test | 7 |  | `TestActionMacros_ActionMacros` | done |

## gpui/tests/eq_hash.rs

Go target: `integration_eq_hash_test.go` (package `gpui_test`), tests in `integration_eq_hash_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `pixels_hash_agrees_with_equality_on_signed_zero` | test | 6 |  | `TestEqHash_PixelsHashAgreesWithEqualityOnSignedZero` | done |
| `render_glyph_params_hash_agrees_with_equality_on_signed_zero` | test | 15 |  | `TestEqHash_RenderGlyphParamsHashAgreesWithEqualityOnSignedZero` | done |

