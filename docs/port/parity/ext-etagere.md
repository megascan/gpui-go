# Parity: ext-etagere

Scope: etagere 0.2.15 (`src/lib.rs`, `src/bucketed.rs`; `src/allocator.rs` is not ported). Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md). External crate: section 7 of PARITY.md.

Status values: `todo`, `done`, `dev:<id>`, `n/a:<id>`. Upstream sources are read from `~/.cargo/registry/src/index.crates.io-1949cf8c6b5b557f/etagere-0.2.15/`.

Notes on the port:

- Geometry is int32 `Point`, `Size` and `Rectangle` (`Min`, `Max`), standing in for euclid's `Point2D`, `Size2D` and `Box2D`.
- `debug_assert!` and `assert!` both become always-on panics with upstream's condition text. Valid use never reaches them, so the difference only shows on misuse.
- `AllocatorOptions` is passed by value where upstream takes `&AllocatorOptions` (it is `Copy`).
- The `checks` feature body is kept behind the constant `checksEnabled = false`, matching upstream's default build where `check()` is a no-op.
- Upstream bug kept as is: `coalesce_shelves` indexes `shelves[shelf_index + i]` before checking the bound, so it panics when the last shelf is empty and too narrow for the request. See docs/port/debt/M0-etagere.md.

## src/lib.rs

Go target: `internal/etagere/lib.go` (package `etagere`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `pub extern crate serde` | use | 78 | serialization | - | n/a:D-ETAGERE-SERDE |
| `pub extern crate euclid` | use | 79 |  | - | dev:D-EUCLID |
| `pub mod ffi` | mod | 84 | ffi | - | n/a:D-ETAGERE-FFI |
| `pub use euclid::{point2, size2}` (`point2`) | use | 88 |  | `Point2` | done |
| `pub use euclid::{point2, size2}` (`size2`) | use | 88 |  | `Size2` | done |
| `Point` | type | 90 |  | `Point` | done |
| `Size` | type | 91 |  | `Size` | done |
| `Rectangle` | type | 92 |  | `Rectangle` | done |
| `AllocatorOptions` | struct | 97 |  | `AllocatorOptions` | done |
| `AllocatorOptions.alignment` | field | 101 |  | `AllocatorOptions.Alignment` | done |
| `AllocatorOptions.vertical_shelves` | field | 105 |  | `AllocatorOptions.VerticalShelves` | done |
| `AllocatorOptions.num_columns` | field | 111 |  | `AllocatorOptions.NumColumns` | done |
| `DEFAULT_OPTIONS` | const | 114 |  | `DefaultOptions` | done |
| `impl Default for AllocatorOptions` | trait-impl | 120 |  | `DefaultAllocatorOptions` | done |
| `Allocation` | struct | 130 |  | `Allocation` | done |
| `Allocation.id` | field | 131 |  | `Allocation.ID` | done |
| `Allocation.rectangle` | field | 132 |  | `Allocation.Rectangle` | done |
| `AllocId` | struct | 139 |  | `AllocID` | done |
| `AllocId::serialize` | method | 158 |  | `(AllocID).Serialize` | done |
| `AllocId::deserialize` | assoc-fn | 163 |  | `AllocIDDeserialize` | done |

## src/bucketed.rs

Go target: `internal/etagere/bucketed.go` (package `etagere`), tests in `bucketed_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `BucketedAtlasAllocator` | struct | 73 |  | `BucketedAtlasAllocator` | done |
| `BucketedAtlasAllocator` (derive Clone) | derive | 71 |  | `(BucketedAtlasAllocator).Clone` | done |
| `BucketedAtlasAllocator::with_options` | assoc-fn | 90 |  | `BucketedAtlasAllocatorWithOptions` | done |
| `BucketedAtlasAllocator::new` | assoc-fn | 120 |  | `NewBucketedAtlasAllocator` | done |
| `BucketedAtlasAllocator::clear` | method | 124 |  | `(BucketedAtlasAllocator).Clear` | done |
| `BucketedAtlasAllocator::size` | method | 133 |  | `(BucketedAtlasAllocator).Size` | done |
| `BucketedAtlasAllocator::grow` | method | 138 |  | `(BucketedAtlasAllocator).Grow` | done |
| `BucketedAtlasAllocator::is_empty` | method | 201 |  | `(BucketedAtlasAllocator).IsEmpty` | done |
| `BucketedAtlasAllocator::allocate` | method | 206 |  | `(BucketedAtlasAllocator).Allocate` | done |
| `BucketedAtlasAllocator::deallocate` | method | 286 |  | `(BucketedAtlasAllocator).Deallocate` | done |
| `BucketedAtlasAllocator::allocated_space` | method | 295 |  | `(BucketedAtlasAllocator).AllocatedSpace` | done |
| `BucketedAtlasAllocator::free_space` | method | 300 |  | `(BucketedAtlasAllocator).FreeSpace` | done |
| `BucketedAtlasAllocator::dump_svg` | method | 572 |  | `(BucketedAtlasAllocator).DumpSvg` | done |
| `BucketedAtlasAllocator::dump_into_svg` | method | 613 |  | `(BucketedAtlasAllocator).DumpIntoSvg` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `atlas_basic` | test | 717 |  | `TestBucketed_AtlasBasic` | done |
| `test_coalesce_shelves` | test | 750 |  | `TestBucketed_CoalesceShelves` | done |
| `grow_vertically` | test | 797 |  | `TestBucketed_GrowVertically` | done |
| `grow_horizontally` | test | 830 |  | `TestBucketed_GrowHorizontally` | done |
| `grow_to_fit_allocation` | test | 863 |  | `TestBucketed_GrowToFitAllocation` | done |
| `columns` | test | 885 |  | `TestBucketed_Columns` | done |
| `vertical` | test | 931 |  | `TestBucketed_Vertical` | done |
| `clear` | test | 960 |  | `TestBucketed_Clear` | done |
| `fuzz_01` | test | 1051 |  | `TestBucketed_Fuzz01` | done |
| `fuzz_02` | test | 1059 |  | `TestBucketed_Fuzz02` | done |
| `fuzz_03` | test | 1066 |  | `TestBucketed_Fuzz03` | done |
| `fuzz_04` | test | 1098 |  | `TestBucketed_Fuzz04` | done |
| `fuzz_05` | test | 1105 |  | `TestBucketed_Fuzz05` | done |

## src/allocator.rs

Not ported: gpui uses only `BucketedAtlasAllocator`. Go target: none.

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `AtlasAllocator` | struct | 70 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::with_options` | assoc-fn | 85 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::new` | assoc-fn | 113 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::clear` | method | 117 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::size` | method | 178 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::allocate` | method | 187 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::deallocate` | method | 393 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::is_empty` | method | 513 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::allocated_space` | method | 529 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::free_space` | method | 534 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::iter` | method | 538 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::get_index` | method | 675 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::get` | method | 685 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::dump_svg` | method | 714 |  | - | n/a:D-ETAGERE-ATLAS |
| `AtlasAllocator::dump_into_svg` | method | 735 |  | - | n/a:D-ETAGERE-ATLAS |
| `Iter` | struct | 834 |  | - | n/a:D-ETAGERE-ATLAS |
| `impl Iterator for Iter` | trait-impl | 839 |  | - | n/a:D-ETAGERE-ATLAS |
| `impl IntoIterator for &AtlasAllocator` | trait-impl | 882 |  | - | n/a:D-ETAGERE-ATLAS |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_simple` | test | 891 |  | - | n/a:D-ETAGERE-ATLAS |
| `test_options` | test | 923 |  | - | n/a:D-ETAGERE-ATLAS |
| `vertical` | test | 975 |  | - | n/a:D-ETAGERE-ATLAS |
| `clear` | test | 1009 |  | - | n/a:D-ETAGERE-ATLAS |
| `fuzz_01` | test | 1101 |  | - | n/a:D-ETAGERE-ATLAS |
| `fuzz_02` | test | 1143 |  | - | n/a:D-ETAGERE-ATLAS |
| `fuzz_03` | test | 1150 |  | - | n/a:D-ETAGERE-ATLAS |
| `fuzz_04` | test | 1182 |  | - | n/a:D-ETAGERE-ATLAS |
| `issue_17_1` | test | 1189 |  | - | n/a:D-ETAGERE-ATLAS |
| `issue_17_2` | test | 1210 |  | - | n/a:D-ETAGERE-ATLAS |
| `issue_35` | test | 1223 |  | - | n/a:D-ETAGERE-ATLAS |
| `bad_fit` | test | 1238 |  | - | n/a:D-ETAGERE-ATLAS |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-EUCLID | euclid `Point2D<i32>`, `Size2D<i32>` and `Box2D<i32>` become the `Point`, `Size` and `Rectangle` structs with the same methods used here; the euclid re-export is not ported | `src/lib.rs:79`, `:90-92` | `internal/etagere/lib.go` |
| D-ETAGERE-ATLAS | `AtlasAllocator` (`src/allocator.rs`) is not ported; gpui uses only `BucketedAtlasAllocator` | `src/allocator.rs` | - |
| D-ETAGERE-FFI | the C ABI module `ffi` is not ported; Go callers use the package directly | `src/lib.rs:84`, `src/ffi.rs` | - |
| D-ETAGERE-SERDE | the `serialization` feature (serde derives) is not ported; `AllocId` serializes through `Serialize` and `AllocIDDeserialize` | `src/lib.rs:78`, `:96`, `:129`, `:138` | `internal/etagere/lib.go` |
