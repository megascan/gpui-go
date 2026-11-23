package collections

import "reflect"

// HashMap is the hash map used across the port (upstream FxHashMap).
type HashMap[K comparable, V any] = map[K]V

// HashSet is the hash set used across the port (upstream FxHashSet).
type HashSet[T comparable] = map[T]struct{}

// IndexMap is a hash map that iterates in insertion order (upstream indexmap::IndexMap).
// The zero value is an empty map ready for use.
type IndexMap[K comparable, V any] struct {
	index  map[K]int
	keys   []K
	values []V
}

// Set inserts or replaces the value under key, keeping the position of an existing key.
func (m *IndexMap[K, V]) Set(key K, value V) {
	if i, ok := m.index[key]; ok {
		m.values[i] = value
		return
	}
	if m.index == nil {
		m.index = make(map[K]int)
	}
	m.index[key] = len(m.keys)
	m.keys = append(m.keys, key)
	m.values = append(m.values, value)
}

// Get returns the value stored under key and whether it exists.
func (m *IndexMap[K, V]) Get(key K) (V, bool) {
	i, ok := m.index[key]
	if !ok {
		var zero V
		return zero, false
	}
	return m.values[i], true
}

// Len returns the number of entries.
func (m *IndexMap[K, V]) Len() int {
	return len(m.keys)
}

// Iter returns the entries in insertion order.
func (m *IndexMap[K, V]) Iter() Iter[K, V] {
	return func(yield func(K, V) bool) {
		for i := range m.keys {
			if !yield(m.keys[i], m.values[i]) {
				return
			}
		}
	}
}

// IndexSet is an insertion-ordered set (upstream indexmap::IndexSet).
type IndexSet[T comparable] = IndexMap[T, struct{}]

// TypeIDHashMap is a map keyed by Go type identity (upstream TypeIdHashMap).
type TypeIDHashMap[V any] = map[reflect.Type]V

// TypeIDHashSet is a set of Go type identities (upstream TypeIdHashSet).
type TypeIDHashSet = map[reflect.Type]struct{}
