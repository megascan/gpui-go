package sumtree

import (
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"testing"
)

func init() {
	treeBase = 2
}

type ctx = struct{}

type u8 uint8

type IntegersSummary struct {
	count        int
	sum          int
	containsEven bool
	max          u8
}

type Count int

type Sum int

type countOfSummary int

type tree8 = SumTree[u8, IntegersSummary, ctx]

func (v u8) Summary(ctx) IntegersSummary {
	return IntegersSummary{count: 1, sum: int(v), containsEven: v&1 == 0, max: v}
}

func (v u8) Key() u8 { return v }

func (u8) Zero(ctx) u8 { return 0 }

func (u8) AddSummary(summary IntegersSummary, _ ctx) u8 { return summary.max }

func (v u8) Cmp(other u8, _ ctx) int { return compareInt(int(v), int(other)) }

func (IntegersSummary) Zero(ctx) IntegersSummary { return IntegersSummary{} }

func (s IntegersSummary) AddSummary(other IntegersSummary, _ ctx) IntegersSummary {
	s.count += other.count
	s.sum += other.sum
	s.containsEven = s.containsEven || other.containsEven
	s.max = max(s.max, other.max)
	return s
}

func (Count) Zero(ctx) Count { return 0 }

func (c Count) AddSummary(summary IntegersSummary, _ ctx) Count { return c + Count(summary.count) }

func (c Count) Cmp(other Count, _ ctx) int { return compareInt(int(c), int(other)) }

func (c countOfSummary) Cmp(cursorLocation IntegersSummary, _ ctx) int {
	return compareInt(int(c), cursorLocation.count)
}

func (Sum) Zero(ctx) Sum { return 0 }

func (s Sum) AddSummary(summary IntegersSummary, _ ctx) Sum { return s + Sum(summary.sum) }

func compareInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	}
	return 0
}

func u8s(values ...int) []u8 {
	out := make([]u8, len(values))
	for i, v := range values {
		out[i] = u8(v)
	}
	return out
}

func u8Range(start, end int) []u8 {
	var out []u8
	for i := start; i < end; i++ {
		out = append(out, u8(i))
	}
	return out
}

func equalItems(t *testing.T, got, want []u8) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !slices.Equal(got, want) {
		t.Fatalf("items = %v, want %v", got, want)
	}
}

func checkItem(t *testing.T, what string, got u8, ok bool, want u8, wantOK bool) {
	t.Helper()
	if ok != wantOK || (ok && got != want) {
		t.Fatalf("%s = (%v, %v), want (%v, %v)", what, got, ok, want, wantOK)
	}
}

func envInt(t *testing.T, name string, def int) int {
	value, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		t.Fatalf("invalid %s variable", name)
	}
	return n
}

func randomU8s(rng *rand.Rand, count int) []u8 {
	out := make([]u8, count)
	for i := range out {
		out[i] = u8(rng.Uint32())
	}
	return out
}

func TestSumTree_ExtendAndPushTree(t *testing.T) {
	var tree1 tree8
	tree1.Extend(u8Range(0, 20), ctx{})

	var tree2 tree8
	tree2.Extend(u8Range(50, 100), ctx{})

	tree1.Append(tree2, ctx{})
	equalItems(t, tree1.Items(ctx{}), append(u8Range(0, 20), u8Range(50, 100)...))
}

func TestSumTree_Random(t *testing.T) {
	startingSeed := envInt(t, "SEED", 0)
	numIterations := envInt(t, "ITERATIONS", 100)
	numOperations := envInt(t, "OPERATIONS", 5)

	for seed := startingSeed; seed < startingSeed+numIterations; seed++ {
		rng := rand.New(rand.NewPCG(uint64(seed), 0))

		var tree tree8
		count := rng.IntN(10)
		if rng.IntN(2) == 1 {
			tree.Extend(randomU8s(rng, count), ctx{})
		} else {
			tree.ParExtend(randomU8s(rng, count), ctx{})
		}

		for range numOperations {
			spliceEnd := rng.IntN(int(tree.Extent[Count](ctx{})) + 1)
			spliceStart := rng.IntN(spliceEnd + 1)
			count := rng.IntN(10)
			treeEnd := tree.Extent[Count](ctx{})
			newItems := randomU8s(rng, count)

			referenceItems := tree.Items(ctx{})
			referenceItems = slices.Replace(slices.Clone(referenceItems), spliceStart, spliceEnd, newItems...)

			cursor := tree.Cursor[Count](ctx{})
			newTree := cursor.Slice(Count(spliceStart), BiasRight)
			if rng.IntN(2) == 1 {
				newTree.Extend(newItems, ctx{})
			} else {
				newTree.ParExtend(newItems, ctx{})
			}
			cursor.Seek(Count(spliceEnd), BiasRight)
			newTree.Append(cursor.Slice(treeEnd, BiasRight), ctx{})
			tree = newTree

			equalItems(t, tree.Items(ctx{}), referenceItems)
			equalItems(t, slices.Collect(tree.Iter()), slices.Collect(tree.Cursor[Unit[IntegersSummary, ctx]](ctx{}).All()))

			filterCursor := tree.Filter[Count](ctx{}, func(summary IntegersSummary) bool { return summary.containsEven })
			type indexed struct {
				index int
				item  u8
			}
			var expectedFilteredItems []indexed
			for i, item := range tree.Items(ctx{}) {
				if item&1 == 0 {
					expectedFilteredItems = append(expectedFilteredItems, indexed{i, item})
				}
			}

			var itemIx int
			if rng.IntN(2) == 1 {
				filterCursor.Next()
				itemIx = 0
			} else {
				filterCursor.Prev()
				itemIx = max(len(expectedFilteredItems)-1, 0)
			}
			for itemIx < len(expectedFilteredItems) {
				actualItem, ok := filterCursor.Item()
				if !ok {
					t.Fatalf("seed %d: filter cursor item is None", seed)
				}
				reference := expectedFilteredItems[itemIx]
				if actualItem != reference.item {
					t.Fatalf("seed %d: filter item %v, want %v", seed, actualItem, reference.item)
				}
				if int(filterCursor.Start()) != reference.index {
					t.Fatalf("seed %d: filter start %v, want %v", seed, filterCursor.Start(), reference.index)
				}
				filterCursor.Next()
				itemIx++

				for itemIx > 0 && rng.Float64() < 0.2 {
					filterCursor.Prev()
					itemIx--

					if itemIx == 0 && rng.Float64() < 0.2 {
						filterCursor.Prev()
						if _, ok := filterCursor.Item(); ok {
							t.Fatalf("seed %d: filter item before start", seed)
						}
						if filterCursor.Start() != 0 {
							t.Fatalf("seed %d: filter start before start = %v", seed, filterCursor.Start())
						}
						filterCursor.Next()
					}
				}
			}
			if _, ok := filterCursor.Item(); ok {
				t.Fatalf("seed %d: filter cursor not at end", seed)
			}

			beforeStart := false
			c := tree.Cursor[Count](ctx{})
			startPos := rng.IntN(len(referenceItems) + 1)
			c.Seek(Count(startPos), BiasRight)
			pos := startPos + rng.IntN(len(referenceItems)-startPos+1)
			c.SeekForward(Count(pos), BiasRight)

			for i := range 10 {
				if int(c.Start()) != pos {
					t.Fatalf("seed %d: start %v, want %v", seed, c.Start(), pos)
				}

				prev, ok := c.PrevItem()
				if pos > 0 {
					checkItem(t, "prev_item", prev, ok, referenceItems[pos-1], true)
				} else {
					checkItem(t, "prev_item", prev, ok, 0, false)
				}

				item, ok := c.Item()
				if pos < len(referenceItems) && !beforeStart {
					checkItem(t, "item", item, ok, referenceItems[pos], true)
				} else {
					checkItem(t, "item", item, ok, 0, false)
				}

				next, ok := c.NextItem()
				if beforeStart {
					if len(referenceItems) > 0 {
						checkItem(t, "next_item", next, ok, referenceItems[0], true)
					} else {
						checkItem(t, "next_item", next, ok, 0, false)
					}
				} else if pos+1 < len(referenceItems) {
					checkItem(t, "next_item", next, ok, referenceItems[pos+1], true)
				} else {
					checkItem(t, "next_item", next, ok, 0, false)
				}

				if i < 5 {
					c.Next()
					if pos < len(referenceItems) {
						pos++
						beforeStart = false
					}
				} else {
					c.Prev()
					if pos == 0 {
						beforeStart = true
					}
					pos = max(pos-1, 0)
				}
			}
		}

		for range 10 {
			end := rng.IntN(int(tree.Extent[Count](ctx{})) + 1)
			start := rng.IntN(end + 1)
			startBias := BiasRight
			if rng.IntN(2) == 1 {
				startBias = BiasLeft
			}
			endBias := BiasRight
			if rng.IntN(2) == 1 {
				endBias = BiasLeft
			}

			cursor := tree.Cursor[Count](ctx{})
			cursor.Seek(Count(start), startBias)
			slice := cursor.Slice(Count(end), endBias)

			cursor.Seek(Count(start), startBias)
			summary := cursor.Summary[Sum](Count(end), endBias)

			if int(summary) != slice.Summary().sum {
				t.Fatalf("seed %d: summary %v, want %v", seed, summary, slice.Summary().sum)
			}
		}
	}
}

func TestSumTree_Cursor(t *testing.T) {
	type cur = Cursor[u8, IntegersSummary, ctx, IntegersSummary]
	check := func(c *cur, item u8, itemOK bool, prev u8, prevOK bool, next u8, nextOK bool, start int) {
		t.Helper()
		got, ok := c.Item()
		checkItem(t, "item", got, ok, item, itemOK)
		got, ok = c.PrevItem()
		checkItem(t, "prev_item", got, ok, prev, prevOK)
		got, ok = c.NextItem()
		checkItem(t, "next_item", got, ok, next, nextOK)
		if c.Start().sum != start {
			t.Fatalf("start.sum = %d, want %d", c.Start().sum, start)
		}
	}

	var tree tree8
	cursor := tree.Cursor[IntegersSummary](ctx{})
	equalItems(t, cursor.Slice(countOfSummary(0), BiasRight).Items(ctx{}), nil)
	check(cursor, 0, false, 0, false, 0, false, 0)
	cursor.Prev()
	check(cursor, 0, false, 0, false, 0, false, 0)
	cursor.Next()
	check(cursor, 0, false, 0, false, 0, false, 0)

	tree = tree8{}
	tree.Extend(u8s(1), ctx{})
	cursor = tree.Cursor[IntegersSummary](ctx{})
	equalItems(t, cursor.Slice(countOfSummary(0), BiasRight).Items(ctx{}), nil)
	check(cursor, 1, true, 0, false, 0, false, 0)

	cursor.Next()
	check(cursor, 0, false, 1, true, 0, false, 1)

	cursor.Prev()
	check(cursor, 1, true, 0, false, 0, false, 0)

	cursor = tree.Cursor[IntegersSummary](ctx{})
	equalItems(t, cursor.Slice(countOfSummary(1), BiasRight).Items(ctx{}), u8s(1))
	check(cursor, 0, false, 1, true, 0, false, 1)

	cursor.Seek(countOfSummary(0), BiasRight)
	equalItems(t, cursor.Slice(countOfSummary(tree.Extent[Count](ctx{})), BiasRight).Items(ctx{}), u8s(1))
	check(cursor, 0, false, 1, true, 0, false, 1)

	tree = tree8{}
	tree.Extend(u8s(1, 2, 3, 4, 5, 6), ctx{})
	cursor = tree.Cursor[IntegersSummary](ctx{})

	equalItems(t, cursor.Slice(countOfSummary(2), BiasRight).Items(ctx{}), u8s(1, 2))
	check(cursor, 3, true, 2, true, 4, true, 3)

	cursor.Next()
	check(cursor, 4, true, 3, true, 5, true, 6)

	cursor.Next()
	check(cursor, 5, true, 4, true, 6, true, 10)

	cursor.Next()
	check(cursor, 6, true, 5, true, 0, false, 15)

	cursor.Next()
	cursor.Next()
	check(cursor, 0, false, 6, true, 0, false, 21)

	cursor.Prev()
	check(cursor, 6, true, 5, true, 0, false, 15)

	cursor.Prev()
	check(cursor, 5, true, 4, true, 6, true, 10)

	cursor.Prev()
	check(cursor, 4, true, 3, true, 5, true, 6)

	cursor.Prev()
	check(cursor, 3, true, 2, true, 4, true, 3)

	cursor.Prev()
	check(cursor, 2, true, 1, true, 3, true, 1)

	cursor.Prev()
	check(cursor, 1, true, 0, false, 2, true, 0)

	cursor.Prev()
	check(cursor, 0, false, 0, false, 1, true, 0)

	cursor.Next()
	check(cursor, 1, true, 0, false, 2, true, 0)

	cursor = tree.Cursor[IntegersSummary](ctx{})
	equalItems(t, cursor.Slice(countOfSummary(tree.Extent[Count](ctx{})), BiasRight).Items(ctx{}), tree.Items(ctx{}))
	check(cursor, 0, false, 6, true, 0, false, 21)

	cursor.Seek(countOfSummary(3), BiasRight)
	equalItems(t, cursor.Slice(countOfSummary(tree.Extent[Count](ctx{})), BiasRight).Items(ctx{}), u8s(4, 5, 6))
	check(cursor, 0, false, 6, true, 0, false, 21)

	cursor.Seek(countOfSummary(1), BiasLeft)
	got, ok := cursor.Item()
	checkItem(t, "item", got, ok, 1, true)
	cursor.Seek(countOfSummary(1), BiasRight)
	got, ok = cursor.Item()
	checkItem(t, "item", got, ok, 2, true)

	cursor.Seek(countOfSummary(1), BiasRight)
	equalItems(t, cursor.Slice(countOfSummary(3), BiasRight).Items(ctx{}), u8s(2, 3))
	equalItems(t, cursor.Slice(countOfSummary(6), BiasLeft).Items(ctx{}), u8s(4, 5))
	equalItems(t, cursor.Slice(countOfSummary(6), BiasRight).Items(ctx{}), u8s(6))
}

func TestSumTree_Edit(t *testing.T) {
	var tree tree8
	ins := EditInsert[u8, u8]
	rem := EditRemove[u8, u8]
	get := func(key u8, want u8, wantOK bool) {
		t.Helper()
		got, ok := tree.Get(key, ctx{})
		checkItem(t, "get", got, ok, want, wantOK)
	}

	removed := tree.Edit([]Edit[u8, u8]{ins(1), ins(2), ins(0)}, ctx{})
	equalItems(t, tree.Items(ctx{}), u8s(0, 1, 2))
	equalItems(t, removed, nil)
	get(0, 0, true)
	get(1, 1, true)
	get(2, 2, true)
	get(4, 0, false)

	removed = tree.Edit([]Edit[u8, u8]{ins(2), ins(4), rem(0)}, ctx{})
	equalItems(t, tree.Items(ctx{}), u8s(1, 2, 4))
	equalItems(t, removed, u8s(0, 2))
	get(0, 0, false)
	get(1, 1, true)
	get(2, 2, true)
	get(4, 4, true)

	tree = SumTreeFromIter[u8](u8s(1, 2, 3, 5, 5, 6, 7, 8), ctx{})
	removed = tree.Edit([]Edit[u8, u8]{ins(5), ins(5)}, ctx{})
	equalItems(t, removed, u8s(5, 5))
	equalItems(t, tree.Items(ctx{}), u8s(1, 2, 3, 5, 5, 6, 7, 8))
}

func TestSumTree_InsertsInPlace(t *testing.T) {
	var check func(tree tree8, isRoot, mayUnderflow bool) uint8
	check = func(tree tree8, isRoot, mayUnderflow bool) uint8 {
		node := tree.root()
		childSummaries := node.childSummaries
		if !(isRoot || mayUnderflow || !node.isUnderflowing()) {
			t.Fatal("node underflows")
		}
		if node.summary != sum(childSummaries, ctx{}) {
			t.Fatal("node summary mismatch")
		}
		if !node.internal {
			if len(node.items) != len(childSummaries) {
				t.Fatal("item summary count mismatch")
			}
			for i, item := range node.items {
				if item.Summary(ctx{}) != childSummaries[i] {
					t.Fatal("item summary mismatch")
				}
			}
			return 0
		}
		for i, childTree := range node.childTrees {
			if childTree.Summary() != childSummaries[i] {
				t.Fatal("child summary mismatch")
			}
			if check(childTree, false, mayUnderflow)+1 != node.height {
				t.Fatal("height mismatch")
			}
		}
		return node.height
	}

	sortedSet := func(m map[u8]bool) []u8 {
		var out []u8
		for k := range m {
			out = append(out, k)
		}
		slices.Sort(out)
		return out
	}

	rng := rand.New(rand.NewPCG(0, 0))
	for range 1000 {
		var tree tree8
		expected := map[u8]bool{}
		type snapshot struct {
			tree     tree8
			expected []u8
		}
		var snapshots []snapshot
		withRemovals := rng.IntN(2) == 1
		for range 1 + rng.IntN(9) {
			start := u8(rng.Uint32())
			width := 255
			if rng.IntN(2) == 1 {
				width = 8
			}
			batchSet := map[u8]bool{}
			for range 1 + rng.IntN(39) {
				batchSet[start+u8(rng.IntN(width+1))] = true
			}
			batch := sortedSet(batchSet)
			remove := withRemovals && rng.IntN(2) == 1
			edits := make([]Edit[u8, u8], len(batch))
			for i, item := range batch {
				if remove {
					edits[i] = EditRemove[u8, u8](item)
				} else {
					edits[i] = EditInsert[u8, u8](item)
				}
			}
			rng.Shuffle(len(edits), func(i, j int) { edits[i], edits[j] = edits[j], edits[i] })

			var replaced []u8
			if len(edits) == 1 && !edits[0].remove {
				if item, ok := tree.InsertOrReplace[u8](edits[0].item, ctx{}); ok {
					replaced = []u8{item}
				}
			} else {
				replaced = tree.Edit(edits, ctx{})
			}
			var intersection []u8
			for _, item := range batch {
				if expected[item] {
					intersection = append(intersection, item)
				}
			}
			equalItems(t, replaced, intersection)
			for _, item := range batch {
				if remove {
					delete(expected, item)
				} else {
					expected[item] = true
				}
			}
			check(tree, true, withRemovals)
			want := sortedSet(expected)
			equalItems(t, slices.Collect(tree.Iter()), want)
			if rng.IntN(2) == 1 {
				snapshots = append(snapshots, snapshot{tree, want})
			}
			for _, s := range snapshots {
				equalItems(t, slices.Collect(s.tree.Iter()), s.expected)
			}
		}
	}
}

func TestSumTree_FromIter(t *testing.T) {
	equalItems(t, SumTreeFromIter[u8](u8Range(0, 100), ctx{}).Items(ctx{}), u8Range(0, 100))

	ix := 0
	var items []u8
	for {
		ix = (ix + 1) % 2
		if ix != 1 {
			break
		}
		items = append(items, 1)
	}
	equalItems(t, SumTreeFromIter[u8](items, ctx{}).Items(ctx{}), u8s(1))
}

func TestSumTree_DimensionsTarget(t *testing.T) {
	type dims = Dimensions[IntegersSummary, ctx, Count, Sum, Unit[IntegersSummary, ctx]]
	tree := SumTreeFromIter[u8](u8s(1, 2, 3, 4, 5, 6), ctx{})
	cursor := tree.Cursor[dims](ctx{})
	cursor.Seek(DimensionsTarget[IntegersSummary, ctx, Count, Sum, Unit[IntegersSummary, ctx]]{Count(3)}, BiasRight)
	if start := cursor.Start(); start.V0 != 3 || start.V1 != 6 {
		t.Fatalf("start = %+v", start)
	}
	got, ok := cursor.Item()
	checkItem(t, "item", got, ok, 4, true)
}

func TestSumTree_PropertySumTree(t *testing.T) {
	gen := PropertySumTree[u8, IntegersSummary](func(r *rand.Rand) u8 { return u8(r.Uint32()) }, 0, 50)
	rng := rand.New(rand.NewPCG(7, 0))
	for range 20 {
		tree := gen(rng)
		if n := len(tree.Items(ctx{})); n >= 50 {
			t.Fatalf("size %d out of range", n)
		}
	}
}
