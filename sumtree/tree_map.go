package sumtree

import (
	"cmp"
	"iter"
)

type unit = struct{}

// TreeMap is a cheaply-copyable ordered map based on a SumTree.
type TreeMap[K cmp.Ordered, V any] struct {
	tree SumTree[MapEntry[K, V], MapKey[K], unit]
}

// MapEntry is one key-value pair of a TreeMap.
type MapEntry[K cmp.Ordered, V any] struct {
	key   K
	value V
}

// MapKey is the summary of a TreeMap subtree: its greatest key, or none.
type MapKey[K cmp.Ordered] struct {
	key K
	ok  bool
}

// MapKeyRef is the key dimension used to seek in a TreeMap.
type MapKeyRef[K cmp.Ordered] struct {
	key K
	ok  bool
}

// TreeSet is an ordered set based on a TreeMap.
type TreeSet[K cmp.Ordered] struct {
	m TreeMap[K, unit]
}

func optionCmp[K cmp.Ordered](a K, aok bool, b K, bok bool) int {
	switch {
	case !aok && !bok:
		return 0
	case !aok:
		return -1
	case !bok:
		return 1
	}
	return cmp.Compare(a, b)
}

// TreeMapFromOrderedEntries builds a map from entries already in key order.
func TreeMapFromOrderedEntries[K cmp.Ordered, V any](entries iter.Seq2[K, V]) TreeMap[K, V] {
	var items []MapEntry[K, V]
	for key, value := range entries {
		items = append(items, MapEntry[K, V]{key, value})
	}
	return TreeMap[K, V]{SumTreeFromIter[MapEntry[K, V]](items, unit{})}
}

// IsEmpty reports whether the map has no entries.
func (m TreeMap[K, V]) IsEmpty() bool {
	return m.tree.IsEmpty()
}

// ContainsKey reports whether key is present.
func (m TreeMap[K, V]) ContainsKey(key K) bool {
	_, ok := m.Get(key)
	return ok
}

// Get returns the value stored under key.
func (m TreeMap[K, V]) Get(key K) (V, bool) {
	_, _, item, ok := m.tree.Find[MapKeyRef[K]](unit{}, MapKeyRef[K]{key, true}, BiasLeft)
	if ok {
		if itemKey := item.Key(); itemKey.ok && itemKey.key == key {
			return item.value, true
		}
	}
	var none V
	return none, false
}

// Insert sets key to value.
func (m *TreeMap[K, V]) Insert(key K, value V) {
	m.tree.InsertOrReplace[MapKey[K]](MapEntry[K, V]{key, value}, unit{})
}

// InsertOrReplace sets key to value and returns the previous value.
func (m *TreeMap[K, V]) InsertOrReplace(key K, value V) (V, bool) {
	old, ok := m.tree.InsertOrReplace[MapKey[K]](MapEntry[K, V]{key, value}, unit{})
	return old.value, ok
}

// Extend inserts every entry, replacing existing keys.
func (m *TreeMap[K, V]) Extend(entries iter.Seq2[K, V]) {
	var edits []Edit[MapEntry[K, V], MapKey[K]]
	for key, value := range entries {
		edits = append(edits, EditInsert[MapEntry[K, V], MapKey[K]](MapEntry[K, V]{key, value}))
	}
	m.tree.Edit(edits, unit{})
}

// Clear removes every entry.
func (m *TreeMap[K, V]) Clear() {
	m.tree = SumTree[MapEntry[K, V], MapKey[K], unit]{}
}

// Remove deletes key and returns its value.
func (m *TreeMap[K, V]) Remove(key K) (V, bool) {
	var removed V
	found := false
	cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
	target := MapKeyRef[K]{key, true}
	newTree := cursor.Slice(target, BiasLeft)
	if target.Cmp(cursor.End(), unit{}) == 0 {
		item, _ := cursor.Item()
		removed, found = item.value, true
		cursor.Next()
	}
	newTree.Append(cursor.Suffix(), unit{})
	m.tree = newTree
	return removed, found
}

// RemoveRange removes the entries from start up to end.
func (m *TreeMap[K, V]) RemoveRange(start MapSeekTarget[K], end MapSeekTarget[K]) {
	startTarget := mapSeekTargetAdaptor[K]{start}
	endTarget := mapSeekTargetAdaptor[K]{end}
	cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
	newTree := cursor.Slice(startTarget, BiasLeft)
	cursor.Seek(endTarget, BiasLeft)
	newTree.Append(cursor.Suffix(), unit{})
	m.tree = newTree
}

// Closest returns the entry with the greatest key less than or equal to key.
func (m TreeMap[K, V]) Closest(key K) (K, V, bool) {
	cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
	cursor.Seek(MapKeyRef[K]{key, true}, BiasRight)
	cursor.Prev()
	item, ok := cursor.Item()
	return item.key, item.value, ok
}

// IterFrom iterates the entries with keys from from onwards.
func (m TreeMap[K, V]) IterFrom(from K) iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
		cursor.Seek(MapKeyRef[K]{from, true}, BiasLeft)
		for entry := range cursor.All() {
			if !yield(entry.key, entry.value) {
				return
			}
		}
	}
}

// Update applies f to the value stored under key and reports whether it was present.
func (m *TreeMap[K, V]) Update(key K, f func(*V)) bool {
	_, ok := m.UpdateR[struct{}](key, func(v *V) struct{} {
		f(v)
		return struct{}{}
	})
	return ok
}

// UpdateR applies f to the value stored under key and returns its result.
func (m *TreeMap[K, V]) UpdateR[R any](key K, f func(*V) R) (R, bool) {
	cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
	target := MapKeyRef[K]{key, true}
	newTree := cursor.Slice(target, BiasLeft)
	var result R
	found := false
	if target.Cmp(cursor.End(), unit{}) == 0 {
		updated, _ := cursor.Item()
		result, found = f(&updated.value), true
		newTree.Push(updated, unit{})
		cursor.Next()
	}
	newTree.Append(cursor.Suffix(), unit{})
	m.tree = newTree
	return result, found
}

// Retain keeps only the entries for which predicate returns true.
func (m *TreeMap[K, V]) Retain(predicate func(K, V) bool) {
	newMap := NewSumTree[MapEntry[K, V]](unit{})

	cursor := m.tree.Cursor[MapKeyRef[K]](unit{})
	cursor.Next()
	for {
		item, ok := cursor.Item()
		if !ok {
			break
		}
		if predicate(item.key, item.value) {
			newMap.Push(item, unit{})
		}
		cursor.Next()
	}

	m.tree = newMap
}

// Iter iterates the entries in key order.
func (m TreeMap[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for entry := range m.tree.Iter() {
			if !yield(entry.key, entry.value) {
				return
			}
		}
	}
}

// Values iterates the values in key order.
func (m TreeMap[K, V]) Values() iter.Seq[V] {
	return func(yield func(V) bool) {
		for entry := range m.tree.Iter() {
			if !yield(entry.value) {
				return
			}
		}
	}
}

// First returns the entry with the smallest key.
func (m TreeMap[K, V]) First() (K, V, bool) {
	entry, ok := m.tree.First()
	return entry.key, entry.value, ok
}

// Last returns the entry with the greatest key.
func (m TreeMap[K, V]) Last() (K, V, bool) {
	entry, ok := m.tree.Last()
	return entry.key, entry.value, ok
}

// InsertTree inserts every entry of other, replacing existing keys.
func (m *TreeMap[K, V]) InsertTree(other TreeMap[K, V]) {
	var edits []Edit[MapEntry[K, V], MapKey[K]]
	for key, value := range other.Iter() {
		edits = append(edits, EditInsert[MapEntry[K, V], MapKey[K]](MapEntry[K, V]{key, value}))
	}
	m.tree.Edit(edits, unit{})
}

type mapSeekTargetAdaptor[K cmp.Ordered] struct {
	target MapSeekTarget[K]
}

func (a mapSeekTargetAdaptor[K]) Cmp(cursorLocation MapKeyRef[K], _ unit) int {
	if cursorLocation.ok {
		return a.target.CmpCursor(cursorLocation.key)
	}
	return 1
}

// MapSeekTarget is a position in a TreeMap's key space.
type MapSeekTarget[K any] interface {
	CmpCursor(cursorLocation K) int
}

// MapSeekKey is a key used as a MapSeekTarget, upstream's blanket impl for `K: Ord`.
type MapSeekKey[K cmp.Ordered] struct {
	Key K
}

// CmpCursor compares the key with cursorLocation.
func (k MapSeekKey[K]) CmpCursor(cursorLocation K) int {
	return cmp.Compare(k.Key, cursorLocation)
}

// Summary returns the entry's key.
func (e MapEntry[K, V]) Summary(unit) MapKey[K] {
	return e.Key()
}

// Key returns the entry's key.
func (e MapEntry[K, V]) Key() MapKey[K] {
	return MapKey[K]{e.key, true}
}

// Zero returns the empty key.
func (MapKey[K]) Zero(unit) MapKey[K] {
	return MapKey[K]{}
}

// AddSummary returns summary, the greatest key so far.
func (MapKey[K]) AddSummary(summary MapKey[K], _ unit) MapKey[K] {
	return summary
}

// Cmp orders keys, the empty key first.
func (k MapKey[K]) Cmp(cursorLocation MapKey[K], _ unit) int {
	return optionCmp(k.key, k.ok, cursorLocation.key, cursorLocation.ok)
}

// Zero returns the empty key.
func (MapKeyRef[K]) Zero(unit) MapKeyRef[K] {
	return MapKeyRef[K]{}
}

// AddSummary returns the key of summary.
func (MapKeyRef[K]) AddSummary(summary MapKey[K], _ unit) MapKeyRef[K] {
	return MapKeyRef[K](summary)
}

// Cmp orders keys, the empty key first.
func (k MapKeyRef[K]) Cmp(cursorLocation MapKeyRef[K], _ unit) int {
	return optionCmp(k.key, k.ok, cursorLocation.key, cursorLocation.ok)
}

// TreeSetFromOrderedEntries builds a set from keys already in order.
func TreeSetFromOrderedEntries[K cmp.Ordered](entries ...K) TreeSet[K] {
	return TreeSet[K]{TreeMapFromOrderedEntries(func(yield func(K, unit) bool) {
		for _, key := range entries {
			if !yield(key, unit{}) {
				return
			}
		}
	})}
}

// IsEmpty reports whether the set has no keys.
func (s TreeSet[K]) IsEmpty() bool {
	return s.m.IsEmpty()
}

// Insert adds key.
func (s *TreeSet[K]) Insert(key K) {
	s.m.Insert(key, unit{})
}

// Remove deletes key and reports whether it was present.
func (s *TreeSet[K]) Remove(key K) bool {
	_, ok := s.m.Remove(key)
	return ok
}

// Extend adds every key.
func (s *TreeSet[K]) Extend(keys ...K) {
	s.m.Extend(func(yield func(K, unit) bool) {
		for _, key := range keys {
			if !yield(key, unit{}) {
				return
			}
		}
	})
}

// Contains reports whether key is present.
func (s TreeSet[K]) Contains(key K) bool {
	_, ok := s.m.Get(key)
	return ok
}

// Iter iterates the keys in order.
func (s TreeSet[K]) Iter() iter.Seq[K] {
	return func(yield func(K) bool) {
		for key := range s.m.Iter() {
			if !yield(key) {
				return
			}
		}
	}
}

// IterFrom iterates the keys from key onwards.
func (s TreeSet[K]) IterFrom(key K) iter.Seq[K] {
	return func(yield func(K) bool) {
		for k := range s.m.IterFrom(key) {
			if !yield(k) {
				return
			}
		}
	}
}
