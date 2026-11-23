# Parity: sumtree

Scope: sum_tree. Milestone: M0. Index and rules: [../PARITY.md](../PARITY.md).

Status values: `todo`, `done`, `dev:<id>` (deviation registered in PARITY.md section 5), `n/a:<id>`. A row's Go symbol is the name derived by the naming rules (PARITY.md section 3); if a builder must deviate, edit the Go column and set `dev:<id>`.

## sum_tree/src/cursor.rs

Go target: `sumtree/cursor.go` (package `sumtree`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `Cursor` | struct | 30 |  | `Cursor` | todo |
| `Iter` | struct | 54 |  | `Iter` | todo |
| `Cursor::new` | assoc-fn | 64 |  | `NewCursor` | todo |
| `Cursor::reset` | method | 75 |  | `(Cursor).Reset` | todo |
| `Cursor::start` | method | 82 |  | `(Cursor).Start` | todo |
| `Cursor::end` | method | 87 |  | `(Cursor).End` | todo |
| `Cursor::item` | method | 99 |  | `(Cursor).Item` | todo |
| `Cursor::item_summary` | method | 118 |  | `(Cursor).ItemSummary` | todo |
| `Cursor::next_item` | method | 139 |  | `(Cursor).NextItem` | todo |
| `Cursor::prev_item` | method | 177 |  | `(Cursor).PrevItem` | todo |
| `Cursor::prev` | method | 216 |  | `(Cursor).Prev` | todo |
| `Cursor::search_backward` | method | 221 |  | `(Cursor).SearchBackward` | todo |
| `Cursor::next` | method | 291 |  | `(Cursor).Next` | todo |
| `Cursor::search_forward` | method | 296 |  | `(Cursor).SearchForward` | todo |
| `Cursor::did_seek` | method | 395 |  | `(Cursor).DidSeek` | todo |
| `Cursor::seek` | method | 408 |  | `(Cursor).Seek` | todo |
| `Cursor::seek_forward` | method | 423 |  | `(Cursor).SeekForward` | todo |
| `Cursor::slice` | method | 432 |  | `(Cursor).Slice` | todo |
| `Cursor::suffix` | method | 447 |  | `(Cursor).Suffix` | todo |
| `Cursor::summary` | method | 452 |  | `(Cursor).Summary` | todo |
| `FilterCursor` | struct | 679 |  | `FilterCursor` | todo |
| `FilterCursor::new` | assoc-fn | 690 |  | `NewFilterCursor` | todo |
| `FilterCursor::start` | method | 702 |  | `(FilterCursor).Start` | todo |
| `FilterCursor::end` | method | 706 |  | `(FilterCursor).End` | todo |
| `FilterCursor::item` | method | 710 |  | `(FilterCursor).Item` | todo |
| `FilterCursor::item_summary` | method | 714 |  | `(FilterCursor).ItemSummary` | todo |
| `FilterCursor::next` | method | 718 |  | `(FilterCursor).Next` | todo |
| `FilterCursor::prev` | method | 722 |  | `(FilterCursor).Prev` | todo |

## sum_tree/src/property_test.rs

Go target: `sumtree/property_testing.go` (package `sumtree`)

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `sum_tree` | fn | 25 |  | `NewSumTree` | todo |

## sum_tree/src/sum_tree.rs

Go target: `sumtree/sum_tree.go` (package `sumtree`), tests in `sum_tree_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TREE_BASE` | const | 18 |  | `TreeBase` | todo |
| `Item` | trait | 34 |  | `Item` | todo |
| `Item::summary` | trait-method | 37 |  | `Item.Summary` | todo |
| `KeyedItem` | trait | 41 |  | `KeyedItem` | todo |
| `KeyedItem::key` | trait-method | 44 |  | `KeyedItem.Key` | todo |
| `Summary` | trait | 51 |  | `Summary` | todo |
| `Summary::zero` | trait-method | 53 |  | `Summary.Zero` | todo |
| `Summary::add_summary` | trait-method | 54 |  | `Summary.AddSummary` | todo |
| `ContextLessSummary` | trait | 57 |  | `ContextLessSummary` | todo |
| `ContextLessSummary::zero` | trait-method | 58 |  | `ContextLessSummary.Zero` | todo |
| `ContextLessSummary::add_summary` | trait-method | 59 |  | `ContextLessSummary.AddSummary` | todo |
| `NoSummary` | struct | 75 |  | `NoSummary` | todo |
| `Dimension` | trait | 95 |  | `Dimension` | todo |
| `Dimension::zero` | trait-method | 96 |  | `Dimension.Zero` | todo |
| `Dimension::add_summary` | trait-method | 98 |  | `Dimension.AddSummary` | todo |
| `Dimension::with_added_summary` | trait-method | 100 |  | `Dimension.WithAddedSummary` | todo |
| `Dimension::from_summary` | trait-method | 105 |  | `Dimension.FromSummary` | todo |
| `SeekTarget` | trait | 122 |  | `SeekTarget` | todo |
| `SeekTarget::cmp` | trait-method | 123 |  | `SeekTarget.Cmp` | todo |
| `Dimensions` | struct | 139 |  | `Dimensions` | todo |
| `Bias` | enum | 189 |  | `Bias` | todo |
| `Bias::invert` | method | 198 |  | `(Bias).Invert` | todo |
| `SumTree` | struct | 213 |  | `SumTree` | todo |
| `SumTree::new` | assoc-fn | 226 |  | `NewSumTree` | todo |
| `SumTree::from_summary` | assoc-fn | 235 |  | `SumTreeFromSummary` | todo |
| `SumTree::from_item` | assoc-fn | 243 |  | `SumTreeFromItem` | todo |
| `SumTree::from_iter` | assoc-fn | 249 |  | `SumTreeFromIter` | todo |
| `SumTree::from_par_iter` | assoc-fn | 318 |  | `SumTreeFromParIter` | todo |
| `SumTree::items` | method | 380 |  | `(SumTree).Items` | todo |
| `SumTree::iter` | method | 391 |  | `(SumTree).Iter` | todo |
| `SumTree::find_exact` | method | 399 |  | `(SumTree).FindExact` | todo |
| `SumTree::find` | method | 425 |  | `(SumTree).Find` | todo |
| `SumTree::find_with_prev` | method | 510 |  | `(SumTree).FindWithPrev` | todo |
| `SumTree::cursor` | method | 596 |  | `(SumTree).Cursor` | todo |
| `SumTree::filter` | method | 608 |  | `(SumTree).Filter` | todo |
| `SumTree::first` | method | 620 |  | `(SumTree).First` | todo |
| `SumTree::last` | method | 624 |  | `(SumTree).Last` | todo |
| `SumTree::last_summary` | method | 628 |  | `(SumTree).LastSummary` | todo |
| `SumTree::update_last` | method | 632 |  | `(SumTree).UpdateLast` | todo |
| `SumTree::update_first` | method | 676 |  | `(SumTree).UpdateFirst` | todo |
| `SumTree::extent` | method | 721 |  | `(SumTree).Extent` | todo |
| `SumTree::summary` | method | 734 |  | `(SumTree).Summary` | todo |
| `SumTree::is_empty` | method | 741 |  | `(SumTree).IsEmpty` | todo |
| `SumTree::extend` | method | 748 |  | `(SumTree).Extend` | todo |
| `SumTree::par_extend` | method | 755 |  | `(SumTree).ParExtend` | todo |
| `SumTree::push` | method | 766 |  | `(SumTree).Push` | todo |
| `SumTree::append` | method | 778 |  | `(SumTree).Append` | todo |
| `SumTree::insert_or_replace` | method | 1148 |  | `(SumTree).InsertOrReplace` | todo |
| `SumTree::remove` | method | 1156 |  | `(SumTree).Remove` | todo |
| `SumTree::edit` | method | 1173 |  | `(SumTree).Edit` | todo |
| `SumTree::get` | method | 1391 |  | `(SumTree).Get` | todo |
| `Node` | enum | 1415 |  | `Node` | todo |
| `Edit` | enum | 1513 |  | `Edit` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_extend_and_push_tree` | test | 1558 |  | `TestSumTree_ExtendAndPushTree` | todo |
| `test_random` | test | 1570 |  | `TestSumTree_Random` | todo |
| `test_cursor` | test | 1747 |  | `TestSumTree_Cursor` | todo |
| `test_edit` | test | 1936 |  | `TestSumTree_Edit` | todo |
| `test_inserts_in_place` | test | 1963 |  | `TestSumTree_InsertsInPlace` | todo |
| `test_from_iter` | test | 2032 |  | `TestSumTree_FromIter` | todo |

## sum_tree/src/tree_map.rs

Go target: `sumtree/tree_map.go` (package `sumtree`), tests in `tree_map_test.go`

| Upstream item | Kind | Line | Cfg | Go symbol | Status |
|---|---|---|---|---|---|
| `TreeMap` | struct | 7 |  | `TreeMap` | todo |
| `MapEntry` | struct | 13 |  | `MapEntry` | todo |
| `MapKey` | struct | 19 |  | `MapKey` | todo |
| `MapKeyRef` | struct | 28 |  | `MapKeyRef` | todo |
| `TreeSet` | struct | 37 |  | `TreeSet` | todo |
| `TreeMap::from_ordered_entries` | assoc-fn | 42 |  | `TreeMapFromOrderedEntries` | todo |
| `TreeMap::is_empty` | method | 52 |  | `(TreeMap).IsEmpty` | todo |
| `TreeMap::contains_key` | method | 56 |  | `(TreeMap).ContainsKey` | todo |
| `TreeMap::get` | method | 60 |  | `(TreeMap).Get` | todo |
| `TreeMap::insert` | method | 75 |  | `(TreeMap).Insert` | todo |
| `TreeMap::insert_or_replace` | method | 79 |  | `(TreeMap).InsertOrReplace` | todo |
| `TreeMap::extend` | method | 85 |  | `(TreeMap).Extend` | todo |
| `TreeMap::clear` | method | 93 |  | `(TreeMap).Clear` | todo |
| `TreeMap::remove` | method | 97 |  | `(TreeMap).Remove` | todo |
| `TreeMap::remove_range` | method | 112 |  | `(TreeMap).RemoveRange` | todo |
| `TreeMap::closest` | method | 124 |  | `(TreeMap).Closest` | todo |
| `TreeMap::iter_from` | method | 132 |  | `(TreeMap).IterFrom` | todo |
| `TreeMap::update` | method | 140 |  | `(TreeMap).Update` | todo |
| `TreeMap::retain` | assoc-fn | 160 |  | `TreeMapRetain` | todo |
| `TreeMap::iter` | method | 176 |  | `(TreeMap).Iter` | todo |
| `TreeMap::values` | method | 180 |  | `(TreeMap).Values` | todo |
| `TreeMap::first` | method | 184 |  | `(TreeMap).First` | todo |
| `TreeMap::last` | method | 188 |  | `(TreeMap).Last` | todo |
| `TreeMap::insert_tree` | method | 192 |  | `(TreeMap).InsertTree` | todo |
| `MapSeekTarget` | trait | 232 |  | `MapSeekTarget` | todo |
| `MapSeekTarget::cmp_cursor` | trait-method | 233 |  | `MapSeekTarget.CmpCursor` | todo |
| `TreeSet::from_ordered_entries` | assoc-fn | 324 |  | `TreeSetFromOrderedEntries` | todo |
| `TreeSet::is_empty` | method | 330 |  | `(TreeSet).IsEmpty` | todo |
| `TreeSet::insert` | method | 334 |  | `(TreeSet).Insert` | todo |
| `TreeSet::remove` | method | 338 |  | `(TreeSet).Remove` | todo |
| `TreeSet::extend` | method | 342 |  | `(TreeSet).Extend` | todo |
| `TreeSet::contains` | method | 346 |  | `(TreeSet).Contains` | todo |
| `TreeSet::iter` | method | 350 |  | `(TreeSet).Iter` | todo |
| `TreeSet::iter_from` | method | 354 |  | `(TreeSet).IterFrom` | todo |

| Upstream test | Attr | Line | Cfg | Go test | Status |
|---|---|---|---|---|---|
| `test_basic` | test | 364 |  | `TestTreeMap_Basic` | todo |
| `test_iter_from` | test | 411 |  | `TestTreeMap_IterFrom` | todo |
| `test_insert_tree` | test | 439 |  | `TestTreeMap_InsertTree` | todo |
| `test_extend` | test | 460 |  | `TestTreeMap_Extend` | todo |
| `test_remove_between_and_path_successor` | test | 474 |  | `TestTreeMap_RemoveBetweenAndPathSuccessor` | todo |

