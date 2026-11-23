# Parity: foundation

Scope: collections, gpui_util, gpui_shared_string, gpui_tokio, refineable. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_util/src/arc_cow.rs

Go target: `-` (not ported, D-STRING)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ArcCow` | enum | 9 |  | `-` | n/a:D-STRING |

## gpui_util/src/lib.rs

Go target: `gpuiutil/lib.go` (package `gpuiutil`), tests in `lib_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `new_std_command` | fn | 22 | windows | `NewStdCommand` | done |
| `new_std_command` | fn | 31 | windows | `NewStdCommand` | done |
| `get_powershell` | fn | 36 | windows | `GetPowershell` | done |
| `get_windows_system_shell` | fn | 145 | windows | `GetWindowsSystemShell` | done |
| `is_utf8_char_boundary` | fn | 158 |  | `IsUTF8CharBoundary` | done |
| `post_inc` | fn | 163 |  | `PostInc` | done |
| `canonical_float_bits` | fn | 173 |  | `CanonicalFloatBits` | done |
| `measure` | fn | 181 |  | `Measure` | done |
| `debug_panic` | macro | 201 |  | `DebugPanic` | done |
| `some_or_debug_panic` | fn | 213 |  | `SomeOrDebugPanic` | done |
| `maybe` | macro | 226 |  | `-` | n/a:D-MACRO |
| `ResultExt` | trait | 237 |  | `ResultExt` | dev:D-LOG |
| `ResultExt::log_err` | trait-method | 240 |  | `ResultExt.LogErr` | done |
| `ResultExt::log_err_with_backtrace` | trait-method | 244 |  | `ResultExt.LogErrWithBacktrace` | done |
| `ResultExt::debug_assert_ok` | trait-method | 248 |  | `ResultExt.DebugAssertOk` | done |
| `ResultExt::warn_on_err` | trait-method | 249 |  | `ResultExt.WarnOnErr` | done |
| `ResultExt::log_with_level` | trait-method | 250 |  | `ResultExt.LogWithLevel` | done |
| `ResultExt::anyhow` | trait-method | 251 |  | `ResultExt.Anyhow` | done |
| `log_err` | fn | 353 |  | `LogErr` | done |
| `TryFutureExt` | trait | 371 |  | `TryFutureExt` | done |
| `TryFutureExt::log_err` | trait-method | 372 |  | `TryFutureExt.LogErr` | done |
| `TryFutureExt::log_tracked_err` | trait-method | 376 |  | `TryFutureExt.LogTrackedErr` | done |
| `TryFutureExt::warn_on_err` | trait-method | 380 |  | `TryFutureExt.WarnOnErr` | done |
| `TryFutureExt::unwrap` | trait-method | 383 |  | `TryFutureExt.Unwrap` | done |
| `TryFutureExtBacktrace` | trait | 390 |  | `TryFutureExtBacktrace` | done |
| `TryFutureExtBacktrace::log_err_with_backtrace` | trait-method | 391 |  | `TryFutureExtBacktrace.LogErrWithBacktrace` | done |
| `TryFutureExtBacktrace::log_tracked_err_with_backtrace` | trait-method | 395 |  | `TryFutureExtBacktrace.LogTrackedErrWithBacktrace` | done |
| `LogErrorFuture` | struct | 467 |  | `LogErrorFuture` | done |
| `LogErrorWithBacktraceFuture` | struct | 494 |  | `LogErrorWithBacktraceFuture` | done |
| `UnwrapFuture` | struct | 524 |  | `UnwrapFuture` | done |
| `Deferred` | struct | 542 |  | `Deferred` | done |
| `Deferred::abort` | method | 546 |  | `(Deferred).Abort` | done |
| `defer` | fn | 561 |  | `Defer` | done |
| `TypeIdHashBuilder` | struct | 566 |  | `TypeIDHashBuilder` | done |
| `TypeIdHasher` | struct | 577 |  | `TypeIDHasher` | done |
| `truncate_to_bottom_n_sorted_by` | fn | 620 |  | `TruncateToBottomNSortedBy` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `type_id_hasher` | test | 604 |  | `TestGpuiUtil_TypeIDHasher` | done |
| `logging_preserves_diagnostics_and_callers` | test | 654 |  | `TestGpuiUtilLoggingTests_LoggingPreservesDiagnosticsAndCallers` | done |

## gpui_shared_string/gpui_shared_string.rs

Go target: `shared_string.go` (package `gpui`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `SharedString` | struct | 15 |  | `SharedString` | todo |
| `SharedString::new_static` | assoc-fn | 40 |  | `NewSharedStringStatic` | todo |
| `SharedString::new` | assoc-fn | 45 |  | `NewSharedString` | todo |
| `SharedString::as_str` | method | 50 |  | `(SharedString).AsStr` | todo |

## gpui_tokio/src/gpui_tokio.rs

Go target: `-` (package `-`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `init` | fn | 12 |  | `Init` | n/a:D-TOKIO |
| `init_from_handle` | fn | 28 |  | `InitFromHandle` | n/a:D-TOKIO |
| `Tokio` | struct | 50 |  | `Tokio` | n/a:D-TOKIO |
| `Tokio::spawn` | assoc-fn | 55 |  | `TokioSpawn` | n/a:D-TOKIO |
| `Tokio::spawn_result` | assoc-fn | 77 |  | `TokioSpawnResult` | n/a:D-TOKIO |
| `Tokio::handle` | assoc-fn | 97 |  | `TokioHandle` | n/a:D-TOKIO |

## refineable/derive_refineable/src/derive_refineable.rs

Go target: `internal/cmd/refinegen/main.go` (package `main`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Refineable` | proc-macro-derive | 10 |  | `Generate` | done |

## refineable/src/refineable.rs

Go target: `refineable/refineable.go` (package `refineable`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Refineable` | trait | 29 |  | `Refineable` | done |
| `Refineable::refine` | trait-method | 38 |  | `Refineable.Refine` | done |
| `Refineable::refined` | trait-method | 42 |  | `Refineable.Refined` | done |
| `Refineable::from_cascade` | trait-method | 45 |  | `FromCascade` | done |
| `Refineable::is_superset_of` | trait-method | 56 |  | `Refineable.IsSupersetOf` | done |
| `Refineable::subtract` | trait-method | 63 |  | `Refineable.Subtract` | done |
| `IsEmpty` | trait | 66 |  | `IsEmpty` | done |
| `IsEmpty::is_empty` | trait-method | 68 |  | `IsEmpty.IsEmpty` | done |
| `Cascade` | struct | 80 |  | `Cascade` | done |
| `CascadeSlot` | struct | 93 |  | `CascadeSlot` | done |
| `Cascade::reserve` | method | 100 |  | `(Cascade).Reserve` | done |
| `Cascade::base` | method | 109 |  | `(Cascade).Base` | done |
| `Cascade::set` | method | 117 |  | `(Cascade).Set` | done |
| `Cascade::merged` | method | 125 |  | `(Cascade).Merged` | done |

## collections/src/collections.rs

Go target: `collections/collections.go` (package `collections`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HashMap` | type | 1 |  | `HashMap` | done |
| `HashSet` | type | 2 |  | `HashSet` | done |
| `IndexMap` | type | 3 |  | `IndexMap` | done |
| `IndexSet` | type | 4 |  | `IndexSet` | done |
| `TypeIdHashMap` | type | 5 |  | `TypeIDHashMap` | done |
| `TypeIdHashSet` | type | 7 |  | `TypeIDHashSet` | done |

## collections/src/vecmap.rs

Go target: `collections/vecmap.go` (package `collections`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VecMap` | struct | 18 |  | `VecMap` | done |
| `VecMap::new` | assoc-fn | 24 |  | `NewVecMap` | done |
| `VecMap::iter` | method | 31 |  | `(VecMap).Iter` | done |
| `VecMap::entry` | method | 39 |  | `(VecMap).Entry` | done |
| `VecMap::entry_ref` | method | 53 |  | `(VecMap).EntryRef` | done |
| `Iter` | struct | 64 |  | `Iter` | done |
| `Entry` | enum | 76 |  | `Entry` | done |
| `Entry::key` | method | 82 |  | `(Entry).Key` | done |
| `Entry::or_insert_with_key` | method | 89 |  | `(Entry).OrInsertWithKey` | done |
| `Entry::or_insert_with` | method | 106 |  | `(Entry).OrInsertWith` | done |
| `Entry::or_insert` | method | 113 |  | `(Entry).OrInsert` | done |
| `Entry::or_insert_default` | method | 117 |  | `(Entry).OrInsertDefault` | done |
| `OccupiedEntry` | struct | 125 |  | `OccupiedEntry` | done |
| `VacantEntry` | struct | 130 |  | `VacantEntry` | done |
| `EntryRef` | enum | 135 |  | `EntryRef` | done |
| `EntryRef::key` | method | 141 |  | `(EntryRef).Key` | done |
| `EntryRef::or_insert_with_key` | method | 153 |  | `(EntryRef).OrInsertWithKey` | done |
| `EntryRef::or_insert_with` | method | 170 |  | `(EntryRef).OrInsertWith` | done |
| `EntryRef::or_insert` | method | 177 |  | `(EntryRef).OrInsert` | done |
| `EntryRef::or_insert_default` | method | 181 |  | `(EntryRef).OrInsertDefault` | done |
| `VacantEntryRef` | struct | 189 |  | `VacantEntryRef` | done |

## collections/src/vecmap_tests.rs

Go target: `collections/vecmap_test.go` (package `collections`), tests in `vecmap_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_entry_vacant_or_insert` | test | 9 |  | `TestVecmap_EntryVacantOrInsert` | done |
| `test_entry_occupied_or_insert_keeps_existing` | test | 17 |  | `TestVecmap_EntryOccupiedOrInsertKeepsExisting` | done |
| `test_entry_or_insert_with` | test | 26 |  | `TestVecmap_EntryOrInsertWith` | done |
| `test_entry_or_insert_with_not_called_when_occupied` | test | 33 |  | `TestVecmap_EntryOrInsertWithNotCalledWhenOccupied` | done |
| `test_entry_or_insert_with_key` | test | 42 |  | `TestVecmap_EntryOrInsertWithKey` | done |
| `test_entry_or_insert_default` | test | 52 |  | `TestVecmap_EntryOrInsertDefault` | done |
| `test_entry_key` | test | 59 |  | `TestVecmap_EntryKey` | done |
| `test_entry_mut_ref_can_be_updated` | test | 67 |  | `TestVecmap_EntryMutRefCanBeUpdated` | done |
| `test_insertion_order_preserved` | test | 75 |  | `TestVecmap_InsertionOrderPreserved` | done |
| `test_multiple_entries_independent` | test | 87 |  | `TestVecmap_MultipleEntriesIndependent` | done |
| `test_entry_ref_vacant_or_insert` | test | 120 |  | `TestVecmap_EntryRefVacantOrInsert` | done |
| `test_entry_ref_occupied_or_insert_keeps_existing` | test | 129 |  | `TestVecmap_EntryRefOccupiedOrInsertKeepsExisting` | done |
| `test_entry_ref_key_not_cloned_when_occupied` | test | 138 |  | `TestVecmap_EntryRefKeyNotClonedWhenOccupied` | done |
| `test_entry_ref_key_cloned_exactly_once_on_vacant_insert` | test | 155 |  | `TestVecmap_EntryRefKeyClonedExactlyOnceOnVacantInsert` | done |
| `test_entry_ref_or_insert_with_key` | test | 168 |  | `TestVecmap_EntryRefOrInsertWithKey` | done |
| `test_entry_ref_or_insert_with_not_called_when_occupied` | test | 179 |  | `TestVecmap_EntryRefOrInsertWithNotCalledWhenOccupied` | done |
| `test_entry_ref_or_insert_default` | test | 189 |  | `TestVecmap_EntryRefOrInsertDefault` | done |
| `test_entry_ref_key` | test | 196 |  | `TestVecmap_EntryRefKey` | done |
| `test_entry_ref_mut_ref_can_be_updated` | test | 205 |  | `TestVecmap_EntryRefMutRefCanBeUpdated` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-LOG | `log` records go through `log/slog` (`slog.Default()`); `log::Level` is `slog.Level`; the record's `target` is `<package>::<file stem>` and `module_path` is `<import path>/<file>` of the Go caller instead of the `crates/<crate>/src/` split; `{:#}` is `err.Error()` and `{:?}` is `%+v`. `debug_assertions` is the `gpuiutil.DebugAssertions` constant (false under `-tags gpui_release`). Futures from `TryFutureExt` are fallible bodies `func() (T, error)` whose wrappers expose `Run` (D-ASYNC). | `gpui_util/src/lib.rs:201-525` | `gpuiutil/lib.go` |
