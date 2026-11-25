# Parity: sumtree

Scope: sum_tree. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## sum_tree/src/cursor.rs

Go target: `sumtree/cursor.go` (package `sumtree`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Cursor` | struct | 30 |  | `Cursor` | done |
| `Iter` | struct | 54 |  | `Iter` | done |
| `Cursor::new` | assoc-fn | 64 |  | `NewCursor` | done |
| `Cursor::reset` | method | 75 |  | `(Cursor).Reset` | done |
| `Cursor::start` | method | 82 |  | `(Cursor).Start` | done |
| `Cursor::end` | method | 87 |  | `(Cursor).End` | done |
| `Cursor::item` | method | 99 |  | `(Cursor).Item` | done |
| `Cursor::item_summary` | method | 118 |  | `(Cursor).ItemSummary` | done |
| `Cursor::next_item` | method | 139 |  | `(Cursor).NextItem` | done |
| `Cursor::prev_item` | method | 177 |  | `(Cursor).PrevItem` | done |
| `Cursor::prev` | method | 216 |  | `(Cursor).Prev` | done |
| `Cursor::search_backward` | method | 221 |  | `(Cursor).SearchBackward` | done |
| `Cursor::next` | method | 291 |  | `(Cursor).Next` | done |
| `Cursor::search_forward` | method | 296 |  | `(Cursor).SearchForward` | done |
| `Cursor::did_seek` | method | 395 |  | `(Cursor).DidSeek` | done |
| `Cursor::seek` | method | 408 |  | `(Cursor).Seek` | done |
| `Cursor::seek_forward` | method | 423 |  | `(Cursor).SeekForward` | done |
| `Cursor::slice` | method | 432 |  | `(Cursor).Slice` | done |
| `Cursor::suffix` | method | 447 |  | `(Cursor).Suffix` | done |
| `Cursor::summary` | method | 452 |  | `(Cursor).Summary` | done |
| `FilterCursor` | struct | 679 |  | `FilterCursor` | done |
| `FilterCursor::new` | assoc-fn | 690 |  | `NewFilterCursor` | done |
| `FilterCursor::start` | method | 702 |  | `(FilterCursor).Start` | done |
| `FilterCursor::end` | method | 706 |  | `(FilterCursor).End` | done |
| `FilterCursor::item` | method | 710 |  | `(FilterCursor).Item` | done |
| `FilterCursor::item_summary` | method | 714 |  | `(FilterCursor).ItemSummary` | done |
| `FilterCursor::next` | method | 718 |  | `(FilterCursor).Next` | done |
| `FilterCursor::prev` | method | 722 |  | `(FilterCursor).Prev` | done |
| `impl Iterator for Iter` | impl | 586 |  | `(Iter).Next` | done |
| `impl Iterator for Cursor` | impl | 659 |  | `(Cursor).All` | done |
| `impl Iterator for FilterCursor` | impl | 727 |  | `(FilterCursor).All` | done |
| `End` | struct | 843 |  | `endTarget` | done |

## sum_tree/src/property_test.rs

Go target: `sumtree/property_testing.go` (package `sumtree`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `sum_tree` | fn | 25 |  | `PropertySumTree` | dev:D-PROPTEST |

## sum_tree/src/sum_tree.rs

Go target: `sumtree/sum_tree.go` (package `sumtree`), tests in `sum_tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TREE_BASE` | const | 18 |  | `TreeBase` | done |
| `Item` | trait | 34 |  | `Item` | done |
| `Item::summary` | trait-method | 37 |  | `Item.Summary` | done |
| `KeyedItem` | trait | 41 |  | `KeyedItem` | done |
| `KeyedItem::key` | trait-method | 44 |  | `KeyedItem.Key` | done |
| `Summary` | trait | 51 |  | `Summary` | done |
| `Summary::zero` | trait-method | 53 |  | `Summary.Zero` | dev:D-SUMVALUE |
| `Summary::add_summary` | trait-method | 54 |  | `Summary.AddSummary` | dev:D-SUMVALUE |
| `ContextLessSummary` | trait | 57 |  | `ContextLessSummary` | dev:D-SUMVALUE |
| `ContextLessSummary::zero` | trait-method | 58 |  | `Summary.Zero` | dev:D-SUMVALUE |
| `ContextLessSummary::add_summary` | trait-method | 59 |  | `Summary.AddSummary` | dev:D-SUMVALUE |
| `NoSummary` | struct | 75 |  | `NoSummary` | done |
| `Dimension` | trait | 95 |  | `Dimension` | done |
| `Dimension::zero` | trait-method | 96 |  | `Dimension.Zero` | dev:D-SUMVALUE |
| `Dimension::add_summary` | trait-method | 98 |  | `Dimension.AddSummary` | dev:D-SUMVALUE |
| `Dimension::with_added_summary` | trait-method | 100 |  | `WithAddedSummary` | dev:D-SUMVALUE |
| `Dimension::from_summary` | trait-method | 105 |  | `FromSummary` | dev:D-SUMVALUE |
| `SeekTarget` | trait | 122 |  | `SeekTarget` | done |
| `SeekTarget::cmp` | trait-method | 123 |  | `SeekTarget.Cmp` | done |
| `Dimensions` | struct | 139 |  | `Dimensions` | done |
| `impl SeekTarget for D: Dimension + Ord` | impl | 126 |  | `OrdDimension` | dev:D-SUMSEEK |
| `impl Dimension for ()` | impl | 132 |  | `Unit` | done |
| `impl SeekTarget<Dimensions> for D1` | impl | 155 |  | `DimensionsTarget` | dev:D-SUMSEEK |
| `Bias` | enum | 189 |  | `Bias` | done |
| `Bias::invert` | method | 198 |  | `(Bias).Invert` | done |
| `SumTree` | struct | 213 |  | `SumTree` | done |
| `SumTree::new` | assoc-fn | 226 |  | `NewSumTree` | done |
| `SumTree::from_summary` | assoc-fn | 235 |  | `SumTreeFromSummary` | done |
| `SumTree::from_item` | assoc-fn | 243 |  | `SumTreeFromItem` | done |
| `SumTree::from_iter` | assoc-fn | 249 |  | `SumTreeFromIter` | done |
| `SumTree::from_par_iter` | assoc-fn | 318 |  | `SumTreeFromParIter` | done |
| `SumTree::items` | method | 380 |  | `(SumTree).Items` | done |
| `SumTree::iter` | method | 391 |  | `(SumTree).Iter` | done |
| `SumTree::find_exact` | method | 399 |  | `(SumTree).FindExact` | done |
| `SumTree::find` | method | 425 |  | `(SumTree).Find` | done |
| `SumTree::find_with_prev` | method | 510 |  | `(SumTree).FindWithPrev` | done |
| `SumTree::cursor` | method | 596 |  | `(SumTree).Cursor` | done |
| `SumTree::filter` | method | 608 |  | `(SumTree).Filter` | done |
| `SumTree::first` | method | 620 |  | `(SumTree).First` | done |
| `SumTree::last` | method | 624 |  | `(SumTree).Last` | done |
| `SumTree::last_summary` | method | 628 |  | `(SumTree).LastSummary` | done |
| `SumTree::update_last` | method | 632 |  | `(SumTree).UpdateLast` | done |
| `SumTree::update_first` | method | 676 |  | `(SumTree).UpdateFirst` | done |
| `SumTree::extent` | method | 721 |  | `(SumTree).Extent` | done |
| `SumTree::summary` | method | 734 |  | `(SumTree).Summary` | done |
| `SumTree::is_empty` | method | 741 |  | `(SumTree).IsEmpty` | done |
| `SumTree::extend` | method | 748 |  | `(SumTree).Extend` | done |
| `SumTree::par_extend` | method | 755 |  | `(SumTree).ParExtend` | done |
| `SumTree::push` | method | 766 |  | `(SumTree).Push` | done |
| `SumTree::append` | method | 778 |  | `(SumTree).Append` | done |
| `SumTree::insert_or_replace` | method | 1148 |  | `(SumTree).InsertOrReplace` | dev:D-COND |
| `SumTree::remove` | method | 1156 |  | `(SumTree).Remove` | dev:D-COND |
| `SumTree::edit` | method | 1173 |  | `(SumTree).Edit` | dev:D-COND |
| `SumTree::get` | method | 1391 |  | `(SumTree).Get` | dev:D-COND |
| `Node` | enum | 1415 |  | `Node` | done |
| `Edit` | enum | 1513 |  | `Edit` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_extend_and_push_tree` | test | 1558 |  | `TestSumTree_ExtendAndPushTree` | done |
| `test_random` | test | 1570 |  | `TestSumTree_Random` | done |
| `test_cursor` | test | 1747 |  | `TestSumTree_Cursor` | done |
| `test_edit` | test | 1936 |  | `TestSumTree_Edit` | done |
| `test_inserts_in_place` | test | 1963 |  | `TestSumTree_InsertsInPlace` | done |
| `test_from_iter` | test | 2032 |  | `TestSumTree_FromIter` | done |

## sum_tree/src/tree_map.rs

Go target: `sumtree/tree_map.go` (package `sumtree`), tests in `tree_map_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TreeMap` | struct | 7 |  | `TreeMap` | dev:D-SUMORD |
| `MapEntry` | struct | 13 |  | `MapEntry` | done |
| `MapKey` | struct | 19 |  | `MapKey` | done |
| `MapKeyRef` | struct | 28 |  | `MapKeyRef` | done |
| `TreeSet` | struct | 37 |  | `TreeSet` | done |
| `TreeMap::from_ordered_entries` | assoc-fn | 42 |  | `TreeMapFromOrderedEntries` | done |
| `TreeMap::is_empty` | method | 52 |  | `(TreeMap).IsEmpty` | done |
| `TreeMap::contains_key` | method | 56 |  | `(TreeMap).ContainsKey` | done |
| `TreeMap::get` | method | 60 |  | `(TreeMap).Get` | done |
| `TreeMap::insert` | method | 75 |  | `(TreeMap).Insert` | done |
| `TreeMap::insert_or_replace` | method | 79 |  | `(TreeMap).InsertOrReplace` | done |
| `TreeMap::extend` | method | 85 |  | `(TreeMap).Extend` | done |
| `TreeMap::clear` | method | 93 |  | `(TreeMap).Clear` | done |
| `TreeMap::remove` | method | 97 |  | `(TreeMap).Remove` | done |
| `TreeMap::remove_range` | method | 112 |  | `(TreeMap).RemoveRange` | done |
| `TreeMap::closest` | method | 124 |  | `(TreeMap).Closest` | done |
| `TreeMap::iter_from` | method | 132 |  | `(TreeMap).IterFrom` | done |
| `TreeMap::update` | method | 140 |  | `(TreeMap).Update` | done |
| `TreeMap::retain` | assoc-fn | 160 |  | `(TreeMap).Retain` | done |
| `TreeMap::iter` | method | 176 |  | `(TreeMap).Iter` | done |
| `TreeMap::values` | method | 180 |  | `(TreeMap).Values` | done |
| `TreeMap::first` | method | 184 |  | `(TreeMap).First` | done |
| `TreeMap::last` | method | 188 |  | `(TreeMap).Last` | done |
| `TreeMap::insert_tree` | method | 192 |  | `(TreeMap).InsertTree` | done |
| `MapSeekTarget` | trait | 232 |  | `MapSeekTarget` | done |
| `MapSeekTarget::cmp_cursor` | trait-method | 233 |  | `MapSeekTarget.CmpCursor` | done |
| `impl MapSeekTarget for K: Ord` | impl | 236 |  | `MapSeekKey` | dev:D-SUMSEEK |
| `TreeSet::from_ordered_entries` | assoc-fn | 324 |  | `TreeSetFromOrderedEntries` | done |
| `TreeSet::is_empty` | method | 330 |  | `(TreeSet).IsEmpty` | done |
| `TreeSet::insert` | method | 334 |  | `(TreeSet).Insert` | done |
| `TreeSet::remove` | method | 338 |  | `(TreeSet).Remove` | done |
| `TreeSet::extend` | method | 342 |  | `(TreeSet).Extend` | done |
| `TreeSet::contains` | method | 346 |  | `(TreeSet).Contains` | done |
| `TreeSet::iter` | method | 350 |  | `(TreeSet).Iter` | done |
| `TreeSet::iter_from` | method | 354 |  | `(TreeSet).IterFrom` | done |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_basic` | test | 364 |  | `TestTreeMap_Basic` | done |
| `test_iter_from` | test | 411 |  | `TestTreeMap_IterFrom` | done |
| `test_insert_tree` | test | 439 |  | `TestTreeMap_InsertTree` | done |
| `test_extend` | test | 460 |  | `TestTreeMap_Extend` | done |
| `test_remove_between_and_path_successor` | test | 474 |  | `TestTreeMap_RemoveBetweenAndPathSuccessor` | done |

## Deviations registered here

| Id | Deviation | Upstream | Go |
|---|---|---|---|
| D-SUMVALUE | `add_summary(&mut self)` becomes a value method returning the updated value, `zero` is called on the zero value of the type; `with_added_summary` / `from_summary` default methods are free functions; `ContextLessSummary` is `Summary` with context `struct{}` (no blanket impl in Go) | `sum_tree/src/sum_tree.rs:51-120` | `sumtree/sum_tree.go` |
| D-SUMSEEK | Go has no blanket impls: every key dimension implements `Cmp` itself (`OrdDimension`), a `Dimensions` cursor seeks by its first dimension through `DimensionsTarget`, and an ordered key seeks a `TreeMap` through `MapSeekKey` | `sum_tree/src/sum_tree.rs:126,155`, `sum_tree/src/tree_map.rs:236` | `sumtree/sum_tree.go`, `sumtree/tree_map.go` |
| D-SUMORD | `TreeMap` / `TreeSet` keys are `cmp.Ordered` instead of any `Ord` type | `sum_tree/src/tree_map.rs:7` | `sumtree/tree_map.go` |
