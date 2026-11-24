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

type deepStatefulTree struct {
	items int
	depth int
}

func (v *deepStatefulTree) Render(_ *Window, _ *Context[deepStatefulTree]) IntoElement {
	children := make([]IntoElement, 0, v.items)
	for item := range v.items {
		var element IntoElement = NewDiv().ID(ElementIDNamedInteger("leaf", uint64(item))).Size(Px(2.))
		for level := v.depth - 1; level >= 0; level-- {
			element = NewDiv().ID(ElementIDNamedInteger("some-container-element", uint64(level))).Child(element)
		}
		children = append(children, NewDiv().ID(ElementIDNamedInteger("item", uint64(item))).Child(element).Size(Px(4.)))
	}
	return NewDiv().Flex().FlexWrap().SizeFull().Children(children...)
}

func TestElementGlobalElementIDTests_FrameCost(t *testing.T) {
	t.Skip("prints timings; run by hand")
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		for _, shape := range [][2]int{{500, 4}, {500, 12}, {2000, 12}} {
			items, depth := shape[0], shape[1]
			window := cx.AddWindow(func(_ *Window, _ *Context[deepStatefulTree]) *deepStatefulTree {
				return &deepStatefulTree{items: items, depth: depth}
			}).AnyWindowHandle
			frame := func() time.Duration {
				var elapsed time.Duration
				divTestUpdateWindow(cx, window, func(window *Window, cx *App) {
					window.Refresh()
					started := time.Now()
					window.Draw(cx).Clear(cx)
					elapsed = time.Since(started)
				})
				return elapsed
			}
			for range 5 {
				frame()
			}
			samples := make([]time.Duration, 0, 40)
			for range 40 {
				samples = append(samples, frame())
			}
			slices.Sort(samples)
			median := samples[len(samples)/2]
			stateful := items * (depth + 2)
			t.Logf("%d items x %d deep (%d stateful elements): median frame %.2f ms, %.0f ns per stateful element",
				items, depth, stateful, median.Seconds()*1e3, float64(median.Nanoseconds())/float64(stateful))
		}
	})
}
