# Rescan of `_ref/`

`go run ./internal/cmd/paritycheck -rescan [-area a,b]` reimplements the PARITY.md section 9 scanner (`internal/cmd/paritycheck/scan`). It skips with a message when `_ref/` is absent (CI).

## Result at zed@6501df1

| | Files | LOC | Public items | Tests |
|---|---|---|---|---|
| Rescan | 300 | 190679 | 6706 | 869 |
| PARITY.md section 2 | 300 | 190679 | 6706 | 869 |

Every per-crate row of section 2 is reproduced exactly; there are no differences to explain. Totals include the 3099 Styled methods expanded from `gpui_macros/src/styles.rs`, compared by name with `styled-methods.tsv`.

## Rules recovered from the totals

- Only `#[cfg(test)]` and `#[cfg(all(..., test, ...))]` (with `test` as a direct argument of `all`) make an item test-only. `any(test, feature = "test-support")` items are public items (they have rows).
- Inherent impl names keep type arguments only when the impl has no generic parameters and every argument is a plain path: `impl Point<Pixels>` gives `Point<Pixels>::scale`, `impl<T> Point<T>` gives `Point::x`, `impl SpringAnimation<()>` and `impl Task<Box<dyn Any>>` give `SpringAnimation::new`, `Task::downcast`.
- An impl's methods are dropped when its type is declared only non-`pub` in the crate.
- `pub trait` declarations count as items, besides their fns (`trait-method`).
- `#[proc_macro_derive(Name)]` is named by the derive (`Action`), in any crate (`refineable`'s `Refineable` too).

## What -rescan reports

- Errors: scanned items or tests of the selected areas' files with no row (matched by file heading, name and test flag, as a multiset), rows of a scanned kind whose name is not declared in its upstream file, headings that name a missing upstream file, and expanded style methods absent from `styled-methods.tsv`. `ext-*` areas are external ports and are not compared.
- Rows the original scan never produced (fields, trait impls, `use`, private items added by owning stages such as `sum_tree` `End`) are not errors.
- Warnings (the section 9 blind spots): 18 `actions!` invocations, 4 `pub` `bitflags!` types (`DisplayChanges`, `WindowingModes`, `Anchor`, `PopupConstraintAdjustment`), and 6 tests inside `proptest!` (`profiler/hang.rs`, `profiler/journal.rs`). Owning stages cover these by reading the file.
