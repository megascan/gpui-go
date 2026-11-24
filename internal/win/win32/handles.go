//go:build windows

package win32

import (
	"sync"
	"sync/atomic"
)

// Handles maps integer keys to Go values, so a value can cross a native callback's context
// parameter without native code holding a Go pointer. Key 0 is never used.
type Handles[T any] struct {
	mu   sync.Mutex
	next atomic.Uintptr
	m    map[uintptr]T
}

// Insert stores v and returns its key.
func (h *Handles[T]) Insert(v T) uintptr {
	key := h.next.Add(1)
	h.mu.Lock()
	if h.m == nil {
		h.m = map[uintptr]T{}
	}
	h.m[key] = v
	h.mu.Unlock()
	return key
}

// Take removes and returns the value stored under key.
func (h *Handles[T]) Take(key uintptr) (T, bool) {
	h.mu.Lock()
	v, ok := h.m[key]
	delete(h.m, key)
	h.mu.Unlock()
	return v, ok
}

// Get returns the value stored under key without removing it.
func (h *Handles[T]) Get(key uintptr) (T, bool) {
	h.mu.Lock()
	v, ok := h.m[key]
	h.mu.Unlock()
	return v, ok
}

// Len returns the number of stored values.
func (h *Handles[T]) Len() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.m)
}
