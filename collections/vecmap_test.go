package collections

import (
	"slices"
	"testing"
)

type pair[K any, V any] struct {
	key   K
	value V
}

func pairs[K comparable, V any](m *VecMap[K, V]) []pair[K, V] {
	var out []pair[K, V]
	for k, v := range m.Iter() {
		out = append(out, pair[K, V]{k, v})
	}
	return out
}

func TestVecmap_EntryVacantOrInsert(t *testing.T) {
	m := NewVecMap[string, int]()
	value := m.Entry("a").OrInsert(1)
	if *value != 1 {
		t.Fatalf("value = %d, want 1", *value)
	}
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 1}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryOccupiedOrInsertKeepsExisting(t *testing.T) {
	m := NewVecMap[string, int]()
	m.Entry("a").OrInsert(1)
	value := m.Entry("a").OrInsert(99)
	if *value != 1 {
		t.Fatalf("value = %d, want 1", *value)
	}
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 1}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryOrInsertWith(t *testing.T) {
	m := NewVecMap[string, int]()
	m.Entry("a").OrInsertWith(func() int { return 42 })
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 42}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryOrInsertWithNotCalledWhenOccupied(t *testing.T) {
	m := NewVecMap[string, int]()
	m.Entry("a").OrInsert(1)
	m.Entry("a").OrInsertWith(func() int { panic("should not be called") })
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 1}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryOrInsertWithKey(t *testing.T) {
	m := NewVecMap[string, string]()
	m.Entry("hello").OrInsertWithKey(func(k string) string { return "HELLO" })
	if got := pairs(m); !slices.Equal(got, []pair[string, string]{{"hello", "HELLO"}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryOrInsertDefault(t *testing.T) {
	m := NewVecMap[string, int]()
	m.Entry("a").OrInsertDefault()
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 0}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryKey(t *testing.T) {
	m := NewVecMap[string, int]()
	if k := m.Entry("a").Key(); k != "a" {
		t.Fatalf("key = %q, want a", k)
	}
	m.Entry("a").OrInsert(1)
	if k := m.Entry("a").Key(); k != "a" {
		t.Fatalf("key = %q, want a", k)
	}
}

func TestVecmap_EntryMutRefCanBeUpdated(t *testing.T) {
	m := NewVecMap[string, int]()
	value := m.Entry("a").OrInsert(0)
	*value = 5
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 5}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_InsertionOrderPreserved(t *testing.T) {
	m := NewVecMap[string, int]()
	m.Entry("b").OrInsert(2)
	m.Entry("a").OrInsert(1)
	m.Entry("c").OrInsert(3)
	want := []pair[string, int]{{"b", 2}, {"a", 1}, {"c", 3}}
	if got := pairs(m); !slices.Equal(got, want) {
		t.Fatalf("entries = %v, want %v", got, want)
	}
}

func TestVecmap_MultipleEntriesIndependent(t *testing.T) {
	m := NewVecMap[int, int]()
	m.Entry(1).OrInsert(10)
	m.Entry(2).OrInsert(20)
	m.Entry(3).OrInsert(30)
	if n := len(pairs(m)); n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
	m.Entry(1).OrInsert(99)
	if n := len(pairs(m)); n != 3 {
		t.Fatalf("count = %d, want 3", n)
	}
}

type countedKey struct {
	value  string
	clones *int
}

func (k countedKey) Clone() countedKey {
	*k.clones++
	return countedKey{value: k.value, clones: k.clones}
}

func TestVecmap_EntryRefVacantOrInsert(t *testing.T) {
	m := NewVecMap[string, int]()
	key := "a"
	value := m.EntryRef(&key).OrInsert(1)
	if *value != 1 {
		t.Fatalf("value = %d, want 1", *value)
	}
	if n := len(pairs(m)); n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
}

func TestVecmap_EntryRefOccupiedOrInsertKeepsExisting(t *testing.T) {
	m := NewVecMap[string, int]()
	m.EntryRef(new("a")).OrInsert(1)
	value := m.EntryRef(new("a")).OrInsert(99)
	if *value != 1 {
		t.Fatalf("value = %d, want 1", *value)
	}
	if n := len(pairs(m)); n != 1 {
		t.Fatalf("count = %d, want 1", n)
	}
}

func TestVecmap_EntryRefKeyNotClonedWhenOccupied(t *testing.T) {
	clones := 0
	key := countedKey{value: "a", clones: &clones}
	m := NewVecMap[countedKey, int]()
	m.EntryRef(&key).OrInsert(1)
	after := clones
	m.EntryRef(&key).OrInsert(99)
	if clones != after {
		t.Fatalf("clones = %d after second lookup, want %d", clones, after)
	}
}

func TestVecmap_EntryRefKeyClonedExactlyOnceOnVacantInsert(t *testing.T) {
	clones := 0
	key := countedKey{value: "a", clones: &clones}
	m := NewVecMap[countedKey, int]()
	m.EntryRef(&key).OrInsert(1)
	if clones != 1 {
		t.Fatalf("clones = %d, want 1", clones)
	}
}

func TestVecmap_EntryRefOrInsertWithKey(t *testing.T) {
	m := NewVecMap[string, string]()
	key := "hello"
	m.EntryRef(&key).OrInsertWithKey(func(k string) string { return "HELLO" })
	if got := pairs(m); !slices.Equal(got, []pair[string, string]{{"hello", "HELLO"}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryRefOrInsertWithNotCalledWhenOccupied(t *testing.T) {
	m := NewVecMap[string, int]()
	key := "a"
	m.EntryRef(&key).OrInsert(1)
	m.EntryRef(&key).OrInsertWith(func() int { panic("should not be called") })
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 1}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryRefOrInsertDefault(t *testing.T) {
	m := NewVecMap[string, int]()
	m.EntryRef(new("a")).OrInsertDefault()
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 0}}) {
		t.Fatalf("entries = %v", got)
	}
}

func TestVecmap_EntryRefKey(t *testing.T) {
	m := NewVecMap[string, int]()
	key := "a"
	if k := m.EntryRef(&key).Key(); k != key {
		t.Fatalf("key = %q, want %q", k, key)
	}
	m.EntryRef(&key).OrInsert(1)
	if k := m.EntryRef(&key).Key(); k != key {
		t.Fatalf("key = %q, want %q", k, key)
	}
}

func TestVecmap_EntryRefMutRefCanBeUpdated(t *testing.T) {
	m := NewVecMap[string, int]()
	key := "a"
	value := m.EntryRef(&key).OrInsert(0)
	*value = 5
	if got := pairs(m); !slices.Equal(got, []pair[string, int]{{"a", 5}}) {
		t.Fatalf("entries = %v", got)
	}
}
