# Parity: foundation

Scope: collections, gpui_util, gpui_shared_string, gpui_tokio, refineable. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## gpui_util/src/arc_cow.rs

Go target: `gpuiutil/arc_cow.go` (package `gpuiutil`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `ArcCow` | enum | 9 |  | `ArcCow` | todo |

## gpui_util/src/lib.rs

Go target: `gpuiutil/lib.go` (package `gpuiutil`), tests in `lib_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `new_std_command` | fn | 22 | windows | `NewStdCommand` | todo |
| `new_std_command` | fn | 31 | windows | `NewStdCommand` | todo |
| `get_powershell` | fn | 36 | windows | `GetPowershell` | todo |
| `get_windows_system_shell` | fn | 145 | windows | `GetWindowsSystemShell` | todo |
| `is_utf8_char_boundary` | fn | 158 |  | `IsUTF8CharBoundary` | todo |
| `post_inc` | fn | 163 |  | `PostInc` | todo |
| `canonical_float_bits` | fn | 173 |  | `CanonicalFloatBits` | todo |
| `measure` | fn | 181 |  | `Measure` | todo |
| `debug_panic` | macro | 201 |  | `see DESIGN §9 (debug_panic!)` | todo |
| `some_or_debug_panic` | fn | 213 |  | `SomeOrDebugPanic` | todo |
| `maybe` | macro | 226 |  | `see DESIGN §9 (maybe!)` | todo |
| `ResultExt` | trait | 237 |  | `ResultExt` | todo |
| `ResultExt::log_err` | trait-method | 240 |  | `ResultExt.LogErr` | todo |
| `ResultExt::log_err_with_backtrace` | trait-method | 244 |  | `ResultExt.LogErrWithBacktrace` | todo |
| `ResultExt::debug_assert_ok` | trait-method | 248 |  | `ResultExt.DebugAssertOk` | todo |
| `ResultExt::warn_on_err` | trait-method | 249 |  | `ResultExt.WarnOnErr` | todo |
| `ResultExt::log_with_level` | trait-method | 250 |  | `ResultExt.LogWithLevel` | todo |
| `ResultExt::anyhow` | trait-method | 251 |  | `ResultExt.Anyhow` | todo |
| `log_err` | fn | 353 |  | `LogErr` | todo |
| `TryFutureExt` | trait | 371 |  | `TryFutureExt` | todo |
| `TryFutureExt::log_err` | trait-method | 372 |  | `TryFutureExt.LogErr` | todo |
| `TryFutureExt::log_tracked_err` | trait-method | 376 |  | `TryFutureExt.LogTrackedErr` | todo |
| `TryFutureExt::warn_on_err` | trait-method | 380 |  | `TryFutureExt.WarnOnErr` | todo |
| `TryFutureExt::unwrap` | trait-method | 383 |  | `TryFutureExt.Unwrap` | todo |
| `TryFutureExtBacktrace` | trait | 390 |  | `TryFutureExtBacktrace` | todo |
| `TryFutureExtBacktrace::log_err_with_backtrace` | trait-method | 391 |  | `TryFutureExtBacktrace.LogErrWithBacktrace` | todo |
| `TryFutureExtBacktrace::log_tracked_err_with_backtrace` | trait-method | 395 |  | `TryFutureExtBacktrace.LogTrackedErrWithBacktrace` | todo |
| `LogErrorFuture` | struct | 467 |  | `LogErrorFuture` | todo |
| `LogErrorWithBacktraceFuture` | struct | 494 |  | `LogErrorWithBacktraceFuture` | todo |
| `UnwrapFuture` | struct | 524 |  | `UnwrapFuture` | todo |
| `Deferred` | struct | 542 |  | `Deferred` | todo |
| `Deferred::abort` | method | 546 |  | `(Deferred).Abort` | todo |
| `defer` | fn | 561 |  | `Defer` | todo |
| `TypeIdHashBuilder` | struct | 566 |  | `TypeIDHashBuilder` | todo |
| `TypeIdHasher` | struct | 577 |  | `TypeIDHasher` | todo |
| `truncate_to_bottom_n_sorted_by` | fn | 620 |  | `TruncateToBottomNSortedBy` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `type_id_hasher` | test | 604 |  | `TestGpuiUtil_TypeIDHasher` | todo |
| `logging_preserves_diagnostics_and_callers` | test | 654 |  | `TestGpuiUtilLoggingTests_LoggingPreservesDiagnosticsAndCallers` | todo |

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
| `Refineable` | proc-macro-derive | 10 |  | `see DESIGN §9 (Refineable)` | todo |

## refineable/src/refineable.rs

Go target: `refineable/refineable.go` (package `refineable`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Refineable` | trait | 29 |  | `Refineable` | todo |
| `Refineable::refine` | trait-method | 38 |  | `Refineable.Refine` | todo |
| `Refineable::refined` | trait-method | 42 |  | `Refineable.Refined` | todo |
| `Refineable::from_cascade` | trait-method | 45 |  | `Refineable.FromCascade` | todo |
| `Refineable::is_superset_of` | trait-method | 56 |  | `Refineable.IsSupersetOf` | todo |
| `Refineable::subtract` | trait-method | 63 |  | `Refineable.Subtract` | todo |
| `IsEmpty` | trait | 66 |  | `IsEmpty` | todo |
| `IsEmpty::is_empty` | trait-method | 68 |  | `IsEmpty.IsEmpty` | todo |
| `Cascade` | struct | 80 |  | `Cascade` | todo |
| `CascadeSlot` | struct | 93 |  | `CascadeSlot` | todo |
| `Cascade::reserve` | method | 100 |  | `(Cascade).Reserve` | todo |
| `Cascade::base` | method | 109 |  | `(Cascade).Base` | todo |
| `Cascade::set` | method | 117 |  | `(Cascade).Set` | todo |
| `Cascade::merged` | method | 125 |  | `(Cascade).Merged` | todo |

## collections/src/collections.rs

Go target: `collections/collections.go` (package `collections`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `HashMap` | type | 1 |  | `HashMap` | todo |
| `HashSet` | type | 2 |  | `HashSet` | todo |
| `IndexMap` | type | 3 |  | `IndexMap` | todo |
| `IndexSet` | type | 4 |  | `IndexSet` | todo |
| `TypeIdHashMap` | type | 5 |  | `TypeIDHashMap` | todo |
| `TypeIdHashSet` | type | 7 |  | `TypeIDHashSet` | todo |

## collections/src/vecmap.rs

Go target: `collections/vecmap.go` (package `collections`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `VecMap` | struct | 18 |  | `VecMap` | todo |
| `VecMap::new` | assoc-fn | 24 |  | `NewVecMap` | todo |
| `VecMap::iter` | method | 31 |  | `(VecMap).Iter` | todo |
| `VecMap::entry` | method | 39 |  | `(VecMap).Entry` | todo |
| `VecMap::entry_ref` | method | 53 |  | `(VecMap).EntryRef` | todo |
| `Iter` | struct | 64 |  | `Iter` | todo |
| `Entry` | enum | 76 |  | `Entry` | todo |
| `Entry::key` | method | 82 |  | `(Entry).Key` | todo |
| `Entry::or_insert_with_key` | method | 89 |  | `(Entry).OrInsertWithKey` | todo |
| `Entry::or_insert_with` | method | 106 |  | `(Entry).OrInsertWith` | todo |
| `Entry::or_insert` | method | 113 |  | `(Entry).OrInsert` | todo |
| `Entry::or_insert_default` | method | 117 |  | `(Entry).OrInsertDefault` | todo |
| `OccupiedEntry` | struct | 125 |  | `OccupiedEntry` | todo |
| `VacantEntry` | struct | 130 |  | `VacantEntry` | todo |
| `EntryRef` | enum | 135 |  | `EntryRef` | todo |
| `EntryRef::key` | method | 141 |  | `(EntryRef).Key` | todo |
| `EntryRef::or_insert_with_key` | method | 153 |  | `(EntryRef).OrInsertWithKey` | todo |
| `EntryRef::or_insert_with` | method | 170 |  | `(EntryRef).OrInsertWith` | todo |
| `EntryRef::or_insert` | method | 177 |  | `(EntryRef).OrInsert` | todo |
| `EntryRef::or_insert_default` | method | 181 |  | `(EntryRef).OrInsertDefault` | todo |
| `VacantEntryRef` | struct | 189 |  | `VacantEntryRef` | todo |

## collections/src/vecmap_tests.rs

Go target: `collections/vecmap_test.go` (package `collections`), tests in `vecmap_test.go`

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_entry_vacant_or_insert` | test | 9 |  | `TestVecmap_EntryVacantOrInsert` | todo |
| `test_entry_occupied_or_insert_keeps_existing` | test | 17 |  | `TestVecmap_EntryOccupiedOrInsertKeepsExisting` | todo |
| `test_entry_or_insert_with` | test | 26 |  | `TestVecmap_EntryOrInsertWith` | todo |
| `test_entry_or_insert_with_not_called_when_occupied` | test | 33 |  | `TestVecmap_EntryOrInsertWithNotCalledWhenOccupied` | todo |
| `test_entry_or_insert_with_key` | test | 42 |  | `TestVecmap_EntryOrInsertWithKey` | todo |
| `test_entry_or_insert_default` | test | 52 |  | `TestVecmap_EntryOrInsertDefault` | todo |
| `test_entry_key` | test | 59 |  | `TestVecmap_EntryKey` | todo |
| `test_entry_mut_ref_can_be_updated` | test | 67 |  | `TestVecmap_EntryMutRefCanBeUpdated` | todo |
| `test_insertion_order_preserved` | test | 75 |  | `TestVecmap_InsertionOrderPreserved` | todo |
| `test_multiple_entries_independent` | test | 87 |  | `TestVecmap_MultipleEntriesIndependent` | todo |
| `test_entry_ref_vacant_or_insert` | test | 120 |  | `TestVecmap_EntryRefVacantOrInsert` | todo |
| `test_entry_ref_occupied_or_insert_keeps_existing` | test | 129 |  | `TestVecmap_EntryRefOccupiedOrInsertKeepsExisting` | todo |
| `test_entry_ref_key_not_cloned_when_occupied` | test | 138 |  | `TestVecmap_EntryRefKeyNotClonedWhenOccupied` | todo |
| `test_entry_ref_key_cloned_exactly_once_on_vacant_insert` | test | 155 |  | `TestVecmap_EntryRefKeyClonedExactlyOnceOnVacantInsert` | todo |
| `test_entry_ref_or_insert_with_key` | test | 168 |  | `TestVecmap_EntryRefOrInsertWithKey` | todo |
| `test_entry_ref_or_insert_with_not_called_when_occupied` | test | 179 |  | `TestVecmap_EntryRefOrInsertWithNotCalledWhenOccupied` | todo |
| `test_entry_ref_or_insert_default` | test | 189 |  | `TestVecmap_EntryRefOrInsertDefault` | todo |
| `test_entry_ref_key` | test | 196 |  | `TestVecmap_EntryRefKey` | todo |
| `test_entry_ref_mut_ref_can_be_updated` | test | 205 |  | `TestVecmap_EntryRefMutRefCanBeUpdated` | todo |

