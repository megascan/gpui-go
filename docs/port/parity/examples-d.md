# Parity: examples-d

Scope: gpui: examples (view_example, testing, input, animation). Milestone: M3. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui/examples/animation.rs

Go target: `examples/animation/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `clicking_spring_position_changes_the_target` | gpui::test | 354 | feat:test-support | `TestAnimation_ClickingSpringPositionChangesTheTarget` | done |

## gpui/examples/input.rs

Go target: `examples/input/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `native_selection_and_length_use_utf16` | gpui::test | 857 | feat:test-support | `TestInput_NativeSelectionAndLengthUseUTF16` | done |
| `marked_selection_is_relative_to_inserted_text` | gpui::test | 875 | feat:test-support | `TestInput_MarkedSelectionIsRelativeToInsertedText` | done |
| `long_single_line_selection_uses_one_geometry_fragment` | gpui::test | 898 | feat:test-support | `TestInput_LongSingleLineSelectionUsesOneGeometryFragment` | done |

## gpui/examples/testing.rs

Go target: `examples/testing/main.go` (package `main`), tests in `main_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `basic_testing` | gpui::test | 234 |  | `TestTesting_BasicTesting` | done |
| `test_counter_in_window` | gpui::test | 261 |  | `TestTesting_CounterInWindow` | done |
| `test_async_operations` | gpui::test | 288 |  | `TestTesting_AsyncOperations` | done |
| `test_allow_parking` | gpui::test | 317 |  | `TestTesting_AllowParking` | done |
| `test_counter_random_operations` | gpui::test(iterations = 10) | 336 |  | `TestTesting_CounterRandomOperations` | done |
| `test_app_sync` | gpui::test | 480 |  | `TestTestingDistributedSystems_AppSync` | done |
| `test_random_interleaving` | gpui::test(iterations = 10) | 503 |  | `TestTestingDistributedSystems_RandomInterleaving` | done |

## gpui/examples/view_example/example_tests.rs

Go target: `examples/view_example/example_tests_test.go` (package `main`; the upstream file is test-only, so there is no non-test Go file)

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `typing_updates_the_shared_string` | gpui::test | 77 |  | `TestExample_TypingUpdatesTheSharedString` | done |
| `sibling_inputs_are_isolated` | gpui::test | 87 |  | `TestExample_SiblingInputsAreIsolated` | done |
| `external_writes_clamp_the_cursor` | gpui::test | 99 |  | `TestExample_ExternalWritesClampTheCursor` | done |
| `arrows_move_the_cursor` | gpui::test | 122 |  | `TestExample_ArrowsMoveTheCursor` | done |

