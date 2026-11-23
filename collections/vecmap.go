package collections

import (
	"iter"
	"slices"
)

// VecMap is a map backed by two slices that iterates in insertion order.
// Suited to small key-value stores; the zero value is an empty map ready for use.
type VecMap[K comparable, V any] struct {
	keys   []K
	values []V
}

// NewVecMap returns an empty VecMap.
func NewVecMap[K comparable, V any]() *VecMap[K, V] {
	return &VecMap[K, V]{}
}

// Iter returns the entries in insertion order.
func (m *VecMap[K, V]) Iter() Iter[K, V] {
	return func(yield func(K, V) bool) {
		for i := range m.keys {
			if !yield(m.keys[i], m.values[i]) {
				return
			}
		}
	}
}

// Iter is the iterator returned by VecMap.Iter.
type Iter[K any, V any] iter.Seq2[K, V]

// Entry returns the entry for key, taking ownership of key when it is vacant.
func (m *VecMap[K, V]) Entry(key K) Entry[K, V] {
	if i := slices.Index(m.keys, key); i >= 0 {
		return Entry[K, V]{occupied: &OccupiedEntry[K, V]{key: m.keys[i], value: &m.values[i]}}
	}
	return Entry[K, V]{vacant: &VacantEntry[K, V]{m: m, key: key}}
}

// EntryRef is like Entry but takes its key by reference; the key is copied (via its
// Clone method when it has one) only when a value is inserted under it.
func (m *VecMap[K, V]) EntryRef(key *K) EntryRef[K, V] {
	if i := slices.Index(m.keys, *key); i >= 0 {
		return EntryRef[K, V]{occupied: &OccupiedEntry[K, V]{key: m.keys[i], value: &m.values[i]}}
	}
	return EntryRef[K, V]{vacant: &VacantEntryRef[K, V]{m: m, key: key}}
}

// Entry is a view of one key in a VecMap, occupied or vacant.
type Entry[K comparable, V any] struct {
	occupied *OccupiedEntry[K, V]
	vacant   *VacantEntry[K, V]
}

// Key returns the entry's key.
func (e Entry[K, V]) Key() K {
	if e.occupied != nil {
		return e.occupied.key
	}
	return e.vacant.key
}

// OrInsertWithKey returns the value under the entry's key, inserting f(key) first if vacant.
func (e Entry[K, V]) OrInsertWithKey(f func(K) V) *V {
	if e.occupied != nil {
		return e.occupied.value
	}
	v := f(e.vacant.key)
	e.vacant.m.values = append(e.vacant.m.values, v)
	e.vacant.m.keys = append(e.vacant.m.keys, e.vacant.key)
	return &e.vacant.m.values[len(e.vacant.m.values)-1]
}

// OrInsertWith returns the value under the entry's key, inserting f() first if vacant.
func (e Entry[K, V]) OrInsertWith(f func() V) *V {
	return e.OrInsertWithKey(func(K) V { return f() })
}

// OrInsert returns the value under the entry's key, inserting value first if vacant.
func (e Entry[K, V]) OrInsert(value V) *V {
	return e.OrInsertWithKey(func(K) V { return value })
}

// OrInsertDefault returns the value under the entry's key, inserting the zero value first if vacant.
func (e Entry[K, V]) OrInsertDefault() *V {
	return e.OrInsertWithKey(func(K) V {
		var zero V
		return zero
	})
}

// OccupiedEntry is an entry whose key is present in the map.
type OccupiedEntry[K comparable, V any] struct {
	key   K
	value *V
}

// VacantEntry is an entry whose key is absent from the map and owns its key.
type VacantEntry[K comparable, V any] struct {
	m   *VecMap[K, V]
	key K
}

// EntryRef is a view of one key in a VecMap, occupied or vacant, keyed by reference.
type EntryRef[K comparable, V any] struct {
	occupied *OccupiedEntry[K, V]
	vacant   *VacantEntryRef[K, V]
}

// Key returns the entry's key.
func (e EntryRef[K, V]) Key() K {
	if e.occupied != nil {
		return e.occupied.key
	}
	return *e.vacant.key
}

// OrInsertWithKey returns the value under the entry's key, inserting f(key) first if vacant.
// The stored key is a Clone of the referenced key when K has a Clone() K method.
func (e EntryRef[K, V]) OrInsertWithKey(f func(K) V) *V {
	if e.occupied != nil {
		return e.occupied.value
	}
	v := f(*e.vacant.key)
	e.vacant.m.values = append(e.vacant.m.values, v)
	e.vacant.m.keys = append(e.vacant.m.keys, cloneKey(*e.vacant.key))
	return &e.vacant.m.values[len(e.vacant.m.values)-1]
}

// OrInsertWith returns the value under the entry's key, inserting f() first if vacant.
func (e EntryRef[K, V]) OrInsertWith(f func() V) *V {
	return e.OrInsertWithKey(func(K) V { return f() })
}

// OrInsert returns the value under the entry's key, inserting value first if vacant.
func (e EntryRef[K, V]) OrInsert(value V) *V {
	return e.OrInsertWithKey(func(K) V { return value })
}

// OrInsertDefault returns the value under the entry's key, inserting the zero value first if vacant.
func (e EntryRef[K, V]) OrInsertDefault() *V {
	return e.OrInsertWithKey(func(K) V {
		var zero V
		return zero
	})
}

// VacantEntryRef is a vacant entry that borrows its key until a value is inserted.
type VacantEntryRef[K comparable, V any] struct {
	m   *VecMap[K, V]
	key *K
}

type cloner[K any] interface {
	Clone() K
}

func cloneKey[K any](key K) K {
	if c, ok := any(key).(cloner[K]); ok {
		return c.Clone()
	}
	return key
}
