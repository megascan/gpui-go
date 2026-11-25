package sumtree

import (
	"cmp"
	"slices"
	"strings"
	"testing"
)

type kv[K any, V any] struct {
	k K
	v V
}

func entries[K cmp.Ordered, V any](m TreeMap[K, V]) []kv[K, V] {
	var out []kv[K, V]
	for k, v := range m.Iter() {
		out = append(out, kv[K, V]{k, v})
	}
	return out
}

func checkEntries[K cmp.Ordered, V comparable](t *testing.T, m TreeMap[K, V], want ...kv[K, V]) {
	t.Helper()
	got := entries(m)
	if len(got) != len(want) || (len(got) > 0 && !slices.Equal(got, want)) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func checkGet[K cmp.Ordered, V comparable](t *testing.T, m TreeMap[K, V], key K, want V, wantOK bool) {
	t.Helper()
	got, ok := m.Get(key)
	if ok != wantOK || (ok && got != want) {
		t.Fatalf("get(%v) = (%v, %v), want (%v, %v)", key, got, ok, want, wantOK)
	}
}

func seq2[K any, V any](pairs ...kv[K, V]) func(func(K, V) bool) {
	return func(yield func(K, V) bool) {
		for _, p := range pairs {
			if !yield(p.k, p.v) {
				return
			}
		}
	}
}

func TestTreeMap_Basic(t *testing.T) {
	var m TreeMap[int, string]
	checkEntries(t, m)

	m.Insert(3, "c")
	checkGet(t, m, 3, "c", true)
	checkEntries(t, m, kv[int, string]{3, "c"})

	m.Insert(1, "a")
	checkGet(t, m, 1, "a", true)
	checkEntries(t, m, kv[int, string]{1, "a"}, kv[int, string]{3, "c"})

	m.Insert(2, "b")
	checkGet(t, m, 2, "b", true)
	checkGet(t, m, 1, "a", true)
	checkGet(t, m, 3, "c", true)
	checkEntries(t, m, kv[int, string]{1, "a"}, kv[int, string]{2, "b"}, kv[int, string]{3, "c"})

	closest := func(key int, wantK int, wantV string, wantOK bool) {
		t.Helper()
		k, v, ok := m.Closest(key)
		if ok != wantOK || (ok && (k != wantK || v != wantV)) {
			t.Fatalf("closest(%d) = (%v, %v, %v)", key, k, v, ok)
		}
	}
	closest(0, 0, "", false)
	closest(1, 1, "a", true)
	closest(10, 3, "c", true)

	m.Remove(2)
	checkGet(t, m, 2, "", false)
	checkEntries(t, m, kv[int, string]{1, "a"}, kv[int, string]{3, "c"})

	closest(2, 1, "a", true)

	m.Remove(3)
	checkGet(t, m, 3, "", false)
	checkEntries(t, m, kv[int, string]{1, "a"})

	m.Remove(1)
	checkGet(t, m, 1, "", false)
	checkEntries(t, m)

	m.Insert(4, "d")
	m.Insert(5, "e")
	m.Insert(6, "f")
	m.Retain(func(key int, _ string) bool { return key%2 == 0 })
	checkEntries(t, m, kv[int, string]{4, "d"}, kv[int, string]{6, "f"})
}

func TestTreeMap_IterFrom(t *testing.T) {
	var m TreeMap[string, int]

	m.Insert("a", 1)
	m.Insert("b", 2)
	m.Insert("baa", 3)
	m.Insert("baaab", 4)
	m.Insert("c", 5)

	collect := func(from, prefix string) []string {
		var out []string
		for k := range m.IterFrom(from) {
			if !strings.HasPrefix(k, prefix) {
				break
			}
			out = append(out, k)
		}
		return out
	}

	result := collect("ba", "ba")
	if len(result) != 2 || !slices.Contains(result, "baa") || !slices.Contains(result, "baaab") {
		t.Fatalf("result = %v", result)
	}

	result = collect("c", "c")
	if len(result) != 1 || !slices.Contains(result, "c") {
		t.Fatalf("result = %v", result)
	}
}

func TestTreeMap_InsertTree(t *testing.T) {
	var m TreeMap[string, int]
	m.Insert("a", 1)
	m.Insert("b", 2)
	m.Insert("c", 3)

	var other TreeMap[string, int]
	other.Insert("a", 2)
	other.Insert("b", 2)
	other.Insert("d", 4)

	m.InsertTree(other)

	if n := len(entries(m)); n != 4 {
		t.Fatalf("count = %d", n)
	}
	checkGet(t, m, "a", 2, true)
	checkGet(t, m, "b", 2, true)
	checkGet(t, m, "c", 3, true)
	checkGet(t, m, "d", 4, true)
}

func TestTreeMap_Extend(t *testing.T) {
	var m TreeMap[string, int]
	m.Insert("a", 1)
	m.Insert("b", 2)
	m.Insert("c", 3)
	m.Extend(seq2(kv[string, int]{"a", 2}, kv[string, int]{"b", 2}, kv[string, int]{"d", 4}))
	if n := len(entries(m)); n != 4 {
		t.Fatalf("count = %d", n)
	}
	checkGet(t, m, "a", 2, true)
	checkGet(t, m, "b", 2, true)
	checkGet(t, m, "c", 3, true)
	checkGet(t, m, "d", 4, true)
}

func pathComponents(p string) []string {
	var out []string
	for c := range strings.SplitSeq(p, "/") {
		if c != "" && c != "." {
			out = append(out, c)
		}
	}
	return out
}

func pathStartsWith(p, base string) bool {
	pc, bc := pathComponents(p), pathComponents(base)
	return len(pc) >= len(bc) && slices.Equal(pc[:len(bc)], bc)
}

func pathCmp(a, b string) int {
	return slices.Compare(pathComponents(a), pathComponents(b))
}

type pathDescendants string

func (d pathDescendants) CmpCursor(key string) int {
	if pathStartsWith(key, string(d)) {
		return 1
	}
	return pathCmp(string(d), key)
}

func TestTreeMap_RemoveBetweenAndPathSuccessor(t *testing.T) {
	var m TreeMap[string, int]

	m.Insert("a", 1)
	m.Insert("a/a", 1)
	m.Insert("b", 2)
	m.Insert("b/a/a", 3)
	m.Insert("b/a/a/a/b", 4)
	m.Insert("c", 5)
	m.Insert("c/a", 6)

	m.RemoveRange(MapSeekKey[string]{"b/a"}, pathDescendants("b/a"))

	checkGet(t, m, "a", 1, true)
	checkGet(t, m, "a/a", 1, true)
	checkGet(t, m, "b", 2, true)
	checkGet(t, m, "b/a/a", 0, false)
	checkGet(t, m, "b/a/a/a/b", 0, false)
	checkGet(t, m, "c", 5, true)
	checkGet(t, m, "c/a", 6, true)

	m.RemoveRange(MapSeekKey[string]{"c"}, pathDescendants("c"))

	checkGet(t, m, "a", 1, true)
	checkGet(t, m, "a/a", 1, true)
	checkGet(t, m, "b", 2, true)
	checkGet(t, m, "c", 0, false)
	checkGet(t, m, "c/a", 0, false)

	m.RemoveRange(MapSeekKey[string]{"a"}, pathDescendants("a"))

	checkGet(t, m, "a", 0, false)
	checkGet(t, m, "a/a", 0, false)
	checkGet(t, m, "b", 2, true)

	m.RemoveRange(MapSeekKey[string]{"b"}, pathDescendants("b"))

	checkGet(t, m, "b", 0, false)
}

func TestTreeMap_TreeSet(t *testing.T) {
	s := TreeSetFromOrderedEntries(1, 3, 5)
	s.Insert(2)
	s.Extend(4, 6)
	if !s.Remove(6) || s.Remove(7) || !s.Contains(4) || s.Contains(6) || s.IsEmpty() {
		t.Fatal("tree set operations")
	}
	if got := slices.Collect(s.IterFrom(3)); !slices.Equal(got, []int{3, 4, 5}) {
		t.Fatalf("iter_from = %v", got)
	}
	if got := slices.Collect(s.Iter()); !slices.Equal(got, []int{1, 2, 3, 4, 5}) {
		t.Fatalf("iter = %v", got)
	}
}
