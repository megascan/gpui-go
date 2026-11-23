package taffy

import (
	"testing"

	"github.com/megascan/gpui-go/gpuiutil"
)

func cacheTestInput(width float32) LayoutInput {
	return LayoutInput{
		RunMode:                       RunModeComputeSize,
		SizingMode:                    SizingModeInherentSize,
		Axis:                          RequestedAxisBoth,
		KnownDimensions:               Size[Option[float32]]{Width: some(width), Height: none[float32]()},
		KnownDimensionsAreDefinite:    Size[bool]{Width: true, Height: true},
		ParentSize:                    SizeOptionF32None,
		AvailableSpace:                Size[AvailableSpace]{Width: AvailableSpaceMaxContent, Height: AvailableSpaceMaxContent},
		VerticalMarginsAreCollapsible: LineBoolFalse,
	}
}

func cacheTestOutput(width float32) LayoutOutput {
	return LayoutOutputFromOuterSize(Size[float32]{Width: width, Height: width})
}

func assertCacheGet(t *testing.T, cache *Cache, input LayoutInput, want Option[LayoutOutput]) {
	t.Helper()
	got := cache.Get(&input)
	g, gok := got.Get()
	w, wok := want.Get()
	if gok != wok || (gok && !g.Equal(w)) {
		t.Fatalf("cache.Get = %v, want %v", got, want)
	}
}

func countMeasureEntries(cache *Cache) int {
	n := 0
	for _, e := range cache.measureEntries {
		if e.IsSome() {
			n++
		}
	}
	return n
}

func TestTreeCache_RecentlyUsedMeasureEntriesGetASecondChance(t *testing.T) {
	cache := NewCache()
	for width := range cacheSize {
		in := cacheTestInput(float32(width))
		cache.Store(&in, cacheTestOutput(float32(width)))
	}
	in := cacheTestInput(cacheSize)
	cache.Store(&in, cacheTestOutput(cacheSize))

	assertCacheGet(t, &cache, cacheTestInput(1.0), some(cacheTestOutput(1.0)))
	in = cacheTestInput(cacheSize + 1)
	cache.Store(&in, cacheTestOutput(cacheSize+1))

	assertCacheGet(t, &cache, cacheTestInput(1.0), some(cacheTestOutput(1.0)))
	assertCacheGet(t, &cache, cacheTestInput(2.0), none[LayoutOutput]())
}

func TestTreeCache_StoringAnExistingMeasurementUpdatesItInPlace(t *testing.T) {
	cache := NewCache()
	in1, in2 := cacheTestInput(1.0), cacheTestInput(2.0)
	cache.Store(&in1, cacheTestOutput(1.0))
	cache.Store(&in2, cacheTestOutput(2.0))
	cache.Store(&in1, cacheTestOutput(3.0))

	if n := countMeasureEntries(&cache); n != 2 {
		t.Fatalf("entries = %d, want 2", n)
	}
	assertCacheGet(t, &cache, cacheTestInput(1.0), some(cacheTestOutput(3.0)))
}

func TestTreeCache_MeasurementsWithMarginCollapseMetadataAreNotCached(t *testing.T) {
	cache := NewCache()

	collapseThrough := cacheTestOutput(1.0)
	collapseThrough.MarginsCanCollapseThrough = true
	in1 := cacheTestInput(1.0)
	cache.Store(&in1, collapseThrough)
	assertCacheGet(t, &cache, in1, none[LayoutOutput]())

	carriedMargin := cacheTestOutput(2.0)
	carriedMargin.TopMargin = CollapsibleMarginSetFromMargin(10.0)
	in2 := cacheTestInput(2.0)
	cache.Store(&in2, carriedMargin)
	assertCacheGet(t, &cache, in2, none[LayoutOutput]())
}

func TestTreeCache_RetrievingAMeasurementOnlyMarksItsSlotAsUsed(t *testing.T) {
	cache := NewCache()
	in1, in2 := cacheTestInput(1.0), cacheTestInput(2.0)
	cache.Store(&in1, cacheTestOutput(1.0))
	cache.Store(&in2, cacheTestOutput(2.0))
	cache.recentlyUsedEntries = 0
	entries := cache.measureEntries

	assertCacheGet(t, &cache, in1, some(cacheTestOutput(1.0)))
	if cache.measureEntries != entries {
		t.Fatal("measure entries changed")
	}
	if cache.recentlyUsedEntries == 0 {
		t.Fatal("recentlyUsedEntries is 0")
	}
}

func cacheTestPerformLayoutInput(width, height float32) LayoutInput {
	in := cacheTestInput(width)
	in.RunMode = RunModePerformLayout
	in.KnownDimensions = Size[Option[float32]]{Width: some(width), Height: some(height)}
	in.ParentSize = Size[Option[float32]]{Width: some(width), Height: some(height)}
	return in
}

func cacheTestOtherModes(input LayoutInput) [4]LayoutInput {
	var out [4]LayoutInput
	for i := range out {
		out[i] = input
	}
	out[0].SizingMode = SizingModeContentSize
	out[1].VerticalMarginsAreCollapsible = LineBoolTrue
	out[2].VerticalMarginsAreCollapsible = Line[bool]{Start: true, End: false}
	out[3].VerticalMarginsAreCollapsible = Line[bool]{Start: false, End: true}
	return out
}

func TestTreeCache_CacheModeDistinguishesEveryCombinationOfInputs(t *testing.T) {
	var modes [8]cacheMode
	for index := range modes {
		in := cacheTestInput(1.0)
		if index&1 == 0 {
			in.SizingMode = SizingModeContentSize
		} else {
			in.SizingMode = SizingModeInherentSize
		}
		in.VerticalMarginsAreCollapsible = Line[bool]{Start: index&2 != 0, End: index&4 != 0}
		modes[index] = cacheModeFromLayoutInput(&in)
	}
	for index, mode := range modes {
		for _, prev := range modes[:index] {
			if prev == mode {
				t.Fatalf("mode %d duplicates an earlier mode", index)
			}
		}
	}
}

func TestTreeCache_RetrievingWithADifferentModeMissesAndKeepsTheEntries(t *testing.T) {
	cache := NewCache()
	pl, in1 := cacheTestPerformLayoutInput(100.0, 50.0), cacheTestInput(1.0)
	cache.Store(&pl, cacheTestOutput(1.0))
	cache.Store(&in1, cacheTestOutput(2.0))

	for _, other := range cacheTestOtherModes(pl) {
		assertCacheGet(t, &cache, other, none[LayoutOutput]())
	}
	for _, other := range cacheTestOtherModes(in1) {
		assertCacheGet(t, &cache, other, none[LayoutOutput]())
	}

	assertCacheGet(t, &cache, pl, some(cacheTestOutput(1.0)))
	assertCacheGet(t, &cache, in1, some(cacheTestOutput(2.0)))
}

func TestTreeCache_StoringWithADifferentModeDropsTheExistingEntries(t *testing.T) {
	pl, in1, in4 := cacheTestPerformLayoutInput(100.0, 50.0), cacheTestInput(1.0), cacheTestInput(4.0)
	for _, other := range cacheTestOtherModes(cacheTestInput(3.0)) {
		cache := NewCache()
		cache.Store(&pl, cacheTestOutput(1.0))
		cache.Store(&in1, cacheTestOutput(2.0))

		cache.Store(&other, cacheTestOutput(3.0))
		assertCacheGet(t, &cache, other, some(cacheTestOutput(3.0)))
		if cache.finalLayoutEntry.IsSome() {
			t.Fatal("final layout entry kept")
		}
		if n := countMeasureEntries(&cache); n != 1 {
			t.Fatalf("entries = %d, want 1", n)
		}

		cache.Store(&in4, cacheTestOutput(4.0))
		assertCacheGet(t, &cache, pl, none[LayoutOutput]())
		assertCacheGet(t, &cache, in1, none[LayoutOutput]())
		assertCacheGet(t, &cache, other, none[LayoutOutput]())
		assertCacheGet(t, &cache, in4, some(cacheTestOutput(4.0)))
	}

	for _, other := range cacheTestOtherModes(pl) {
		cache := NewCache()
		cache.Store(&pl, cacheTestOutput(1.0))
		cache.Store(&in1, cacheTestOutput(2.0))

		cache.Store(&other, cacheTestOutput(3.0))
		assertCacheGet(t, &cache, other, some(cacheTestOutput(3.0)))
		if n := countMeasureEntries(&cache); n != 0 {
			t.Fatalf("entries = %d, want 0", n)
		}
		assertCacheGet(t, &cache, pl, none[LayoutOutput]())
	}
}

func TestTreeCache_HiddenLayoutsDoNotAffectTheCache(t *testing.T) {
	cache := NewCache()
	in1 := cacheTestInput(1.0)
	cache.Store(&in1, cacheTestOutput(1.0))
	hidden := LayoutInputHidden
	cache.Store(&hidden, cacheTestOutput(2.0))
	assertCacheGet(t, &cache, in1, some(cacheTestOutput(1.0)))
}

func TestTreeCache_ModeChangeEvictionsAreCountedForNonEmptyCachesOnly(t *testing.T) {
	if !gpuiutil.DebugAssertions {
		t.Skip("needs debug assertions")
	}
	contentSize := cacheTestInput(1.0)
	contentSize.SizingMode = SizingModeContentSize
	evictions := CacheModeChangeEvictions()

	cache := NewCache()
	cache.Store(&contentSize, cacheTestOutput(1.0))
	cs2 := cacheTestInput(2.0)
	cs2.SizingMode = SizingModeContentSize
	cache.Store(&cs2, cacheTestOutput(2.0))
	if got := CacheModeChangeEvictions(); got != evictions {
		t.Fatalf("evictions = %d, want %d", got, evictions)
	}

	in1 := cacheTestInput(1.0)
	cache.Store(&in1, cacheTestOutput(1.0))
	if got := CacheModeChangeEvictions(); got != evictions+1 {
		t.Fatalf("evictions = %d, want %d", got, evictions+1)
	}

	cache.Clear()
	cache.Store(&contentSize, cacheTestOutput(1.0))
	if got := CacheModeChangeEvictions(); got != evictions+1 {
		t.Fatalf("evictions = %d, want %d", got, evictions+1)
	}
}
