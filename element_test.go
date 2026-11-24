package gpui

import (
	"slices"
	"testing"
	"time"
)

func globalElementIDTestPath(depth int, leaf uint64) []ElementID {
	ids := make([]ElementID, 0, depth+1)
	for level := range uint64(depth) {
		ids = append(ids, ElementIDNamedInteger("some-container-element", level))
	}
	ids = append(ids, ElementIDNamedInteger("leaf", leaf))
	return ids
}

func TestElementGlobalElementIDTests_EqualPathsAreEqualAndHashAlike(t *testing.T) {
	a := newGlobalElementID(globalElementIDTestPath(8, 1))
	b := newGlobalElementID(globalElementIDTestPath(8, 1))
	c := newGlobalElementID(globalElementIDTestPath(8, 2))
	shallower := newGlobalElementID(globalElementIDTestPath(7, 1))
	if !a.Equal(b) {
		t.Fatalf("a != b")
	}
	if Hash(a) != Hash(b) {
		t.Fatalf("hash(a) != hash(b)")
	}
	if a.accesskitNodeID() != b.accesskitNodeID() {
		t.Fatalf("accesskit ids differ")
	}
	if a.Equal(c) {
		t.Fatalf("a == c")
	}
	if a.Equal(shallower) {
		t.Fatalf("a == shallower")
	}
	if Hash(a) == Hash(c) {
		t.Fatalf("hash(a) == hash(c)")
	}
	if !slices.Equal(a.IDs(), globalElementIDTestPath(8, 1)) {
		t.Fatalf("ids differ: %v", a.IDs())
	}
	if a.String() != b.String() {
		t.Fatalf("%q != %q", a.String(), b.String())
	}
	if !DefaultGlobalElementID().Equal(newGlobalElementID(nil)) {
		t.Fatalf("default != new(&[])")
	}
}

func TestElementGlobalElementIDTests_HashCost(t *testing.T) {
	t.Skip("prints timings; run by hand")
	ids := globalElementIDTestPath(11, 7)
	id := newGlobalElementID(ids)
	const n = 1_000_000
	var acc uint64
	started := time.Now()
	for range n {
		var h FxHasher
		h.WriteUsize(uint64(len(ids)))
		for _, id := range ids {
			id.Hash(&h)
		}
		acc ^= h.Finish()
	}
	walk := time.Since(started)
	started = time.Now()
	for range n {
		acc ^= Hash(id)
	}
	cached := time.Since(started)
	started = time.Now()
	for range n {
		acc ^= newGlobalElementID(ids).hash
	}
	build := time.Since(started)
	t.Logf("12-deep path: hash by walking ids %.1f ns, cached %.1f ns, build (clone + hash once) %.1f ns [%d]",
		float64(walk.Nanoseconds())/n, float64(cached.Nanoseconds())/n, float64(build.Nanoseconds())/n, acc)
}
