# Parity: core-examples

Scope: gpui: examples, integration tests, benches. Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/benches/focus_cleanup.rs

Go target: `bench_focus_cleanup_test.go` (package `gpui_test`), tests in `bench_focus_cleanup_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `no_drops` | gpui::bench(inputs = [0, 100, 1_000, 10_000], input_name = "handles", group = "focus_cleanup_no_drops_10000_effects", fps = 120) | 13 |  | `BenchmarkFocusCleanup_NoDrops` | todo |
| `final_drops` | gpui::bench(inputs = [10_000], input_name = "handles", group = "focus_cleanup_final_drops_10000_effects", fps = 120) | 23 |  | `BenchmarkFocusCleanup_FinalDrops` | todo |
| `final_drops_with_windows` | gpui::bench(inputs = [1, 10], input_name = "windows", group = "focus_cleanup_final_drops_with_windows_10000_handles_10000_effects", fps = 120) | 33 |  | `BenchmarkFocusCleanup_FinalDropsWithWindows` | todo |

## gpui/examples/animation.rs

Go target: `examples/animation/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `clicking_spring_position_changes_the_target` | gpui::test | 354 | feat:test-support | `TestAnimation_ClickingSpringPositionChangesTheTarget` | todo |

## gpui/examples/input.rs

Go target: `examples/input/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `native_selection_and_length_use_utf16` | gpui::test | 857 | feat:test-support | `TestInput_NativeSelectionAndLengthUseUTF16` | todo |
| `marked_selection_is_relative_to_inserted_text` | gpui::test | 875 | feat:test-support | `TestInput_MarkedSelectionIsRelativeToInsertedText` | todo |
| `long_single_line_selection_uses_one_geometry_fragment` | gpui::test | 898 | feat:test-support | `TestInput_LongSingleLineSelectionUsesOneGeometryFragment` | todo |

## gpui/examples/testing.rs

Go target: `examples/testing/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `basic_testing` | gpui::test | 234 |  | `TestTesting_BasicTesting` | todo |
| `test_counter_in_window` | gpui::test | 261 |  | `TestTesting_CounterInWindow` | todo |
| `test_async_operations` | gpui::test | 288 |  | `TestTesting_AsyncOperations` | todo |
| `test_allow_parking` | gpui::test | 317 |  | `TestTesting_AllowParking` | todo |
| `test_counter_random_operations` | gpui::test(iterations = 10) | 336 |  | `TestTesting_CounterRandomOperations` | todo |
| `test_app_sync` | gpui::test | 480 |  | `TestTestingDistributedSystems_AppSync` | todo |
| `test_random_interleaving` | gpui::test(iterations = 10) | 503 |  | `TestTestingDistributedSystems_RandomInterleaving` | todo |

## gpui/examples/view_example/example_tests.rs

Go target: `examples/view_example/example_tests.go` (package `main`), tests in `example_tests_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `typing_updates_the_shared_string` | gpui::test | 77 |  | `TestExample_TypingUpdatesTheSharedString` | todo |
| `sibling_inputs_are_isolated` | gpui::test | 87 |  | `TestExample_SiblingInputsAreIsolated` | todo |
| `external_writes_clamp_the_cursor` | gpui::test | 99 |  | `TestExample_ExternalWritesClampTheCursor` | todo |
| `arrows_move_the_cursor` | gpui::test | 122 |  | `TestExample_ArrowsMoveTheCursor` | todo |

## gpui/tests/action_macros.rs

Go target: `integration_action_macros_test.go` (package `gpui_test`), tests in `integration_action_macros_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_action_macros` | test | 7 |  | `TestActionMacros_ActionMacros` | todo |

## gpui/tests/eq_hash.rs

Go target: `integration_eq_hash_test.go` (package `gpui_test`), tests in `integration_eq_hash_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `pixels_hash_agrees_with_equality_on_signed_zero` | test | 6 |  | `TestEqHash_PixelsHashAgreesWithEqualityOnSignedZero` | todo |
| `render_glyph_params_hash_agrees_with_equality_on_signed_zero` | test | 15 |  | `TestEqHash_RenderGlyphParamsHashAgreesWithEqualityOnSignedZero` | todo |

