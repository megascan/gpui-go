package etagere

import "testing"

func mustAllocate(t *testing.T, a *BucketedAtlasAllocator, size Size) Allocation {
	t.Helper()
	alloc, ok := a.Allocate(size)
	if !ok {
		t.Fatalf("allocate %v: want a rectangle", size)
	}
	return alloc
}

func noAllocate(t *testing.T, a *BucketedAtlasAllocator, size Size) {
	t.Helper()
	if _, ok := a.Allocate(size); ok {
		t.Fatalf("allocate %v: want none", size)
	}
}

func inRange(val, lo, hi int32) bool {
	return val >= lo && val < hi
}

func TestBucketed_AtlasBasic(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(1000, 1000))

	full := mustAllocate(t, atlas, Size2(1000, 1000)).ID
	noAllocate(t, atlas, Size2(1, 1))

	atlas.Deallocate(full)
	a := mustAllocate(t, atlas, Size2(10, 10)).ID
	b := mustAllocate(t, atlas, Size2(50, 30)).ID
	c := mustAllocate(t, atlas, Size2(12, 45)).ID
	d := mustAllocate(t, atlas, Size2(60, 45)).ID
	e := mustAllocate(t, atlas, Size2(1, 1)).ID
	f := mustAllocate(t, atlas, Size2(128, 128)).ID
	g := mustAllocate(t, atlas, Size2(256, 256)).ID

	atlas.Deallocate(b)
	atlas.Deallocate(f)
	atlas.Deallocate(c)
	atlas.Deallocate(e)
	h := mustAllocate(t, atlas, Size2(500, 200)).ID
	atlas.Deallocate(a)
	i := mustAllocate(t, atlas, Size2(500, 200)).ID
	atlas.Deallocate(g)
	atlas.Deallocate(h)
	atlas.Deallocate(d)
	atlas.Deallocate(i)

	full = mustAllocate(t, atlas, Size2(1000, 1000)).ID
	noAllocate(t, atlas, Size2(1, 1))
	atlas.Deallocate(full)
}

func TestBucketed_CoalesceShelves(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(256, 256))

	var ids []AllocID
	for range 7 {
		for range 8 {
			ids = append(ids, mustAllocate(t, atlas, Size2(32, 32)).ID)
		}
	}

	for i := range 8 {
		atlas.Deallocate(ids[i])
	}

	for i := 16; i < 32; i++ {
		atlas.Deallocate(ids[i])
	}

	noAllocate(t, atlas, Size2(70, 70))

	id := mustAllocate(t, atlas, Size2(64, 64)).ID

	for i := 8; i < 16; i++ {
		atlas.Deallocate(ids[i])
	}

	atlas.Deallocate(id)

	for i := 32; i < 56; i++ {
		atlas.Deallocate(ids[i])
	}

	if !atlas.IsEmpty() {
		t.Fatal("atlas is not empty")
	}
	if atlas.AllocatedSpace() != 0 {
		t.Fatalf("allocated space = %d, want 0", atlas.AllocatedSpace())
	}
}

func TestBucketed_GrowVertically(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(256, 256))

	var ids []AllocID
	for range 7 {
		for range 8 {
			ids = append(ids, mustAllocate(t, atlas, Size2(32, 32)).ID)
		}
	}

	for i := range 8 {
		atlas.Deallocate(ids[i])
	}

	for i := 16; i < 32; i++ {
		atlas.Deallocate(ids[i])
	}

	noAllocate(t, atlas, Size2(70, 70))

	atlas.Grow(Size2(256, 256+70-32))

	mustAllocate(t, atlas, Size2(70, 70))
}

func TestBucketed_GrowHorizontally(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(256, 256))

	var ids []AllocID
	for range 7 {
		for range 8 {
			ids = append(ids, mustAllocate(t, atlas, Size2(32, 32)).ID)
		}
	}

	for i := range 8 {
		atlas.Deallocate(ids[i])
	}

	for i := 16; i < 32; i++ {
		atlas.Deallocate(ids[i])
	}

	noAllocate(t, atlas, Size2(512, 32))

	atlas.Grow(Size2(256*2, 256))

	mustAllocate(t, atlas, Size2(512, 32))
}

func TestBucketed_GrowToFitAllocation(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(32, 32))

	mustAllocate(t, atlas, Size2(32, 32))

	bigAllocation := Size2(256, 256)

	noAllocate(t, atlas, bigAllocation)

	atlas.Grow(Size2(256, 32+256))

	mustAllocate(t, atlas, Size2(32, 32))

	mustAllocate(t, atlas, bigAllocation)
}

func TestBucketed_Columns(t *testing.T) {
	options := DefaultOptions
	options.NumColumns = 2
	atlas := BucketedAtlasAllocatorWithOptions(Size2(64, 64), options)

	a := mustAllocate(t, atlas, Size2(24, 46))
	b := mustAllocate(t, atlas, Size2(24, 32))
	c := mustAllocate(t, atlas, Size2(24, 32))

	if !inRange(a.Rectangle.Min.X, 0, 32) || !inRange(a.Rectangle.Max.X, 0, 32) {
		t.Fatalf("a = %v, want x in [0,32)", a.Rectangle)
	}
	if !inRange(b.Rectangle.Min.X, 32, 64) || !inRange(b.Rectangle.Max.X, 32, 64) {
		t.Fatalf("b = %v, want x in [32,64)", b.Rectangle)
	}
	if !inRange(c.Rectangle.Min.X, 32, 64) || !inRange(c.Rectangle.Max.X, 32, 64) {
		t.Fatalf("c = %v, want x in [32,64)", c.Rectangle)
	}

	atlas.Deallocate(b.ID)
	atlas.Deallocate(c.ID)
	atlas.Deallocate(a.ID)

	if !atlas.IsEmpty() {
		t.Fatal("atlas is not empty")
	}
	if atlas.AllocatedSpace() != 0 {
		t.Fatalf("allocated space = %d, want 0", atlas.AllocatedSpace())
	}

	a = mustAllocate(t, atlas, Size2(24, 46))
	b = mustAllocate(t, atlas, Size2(24, 32))
	c = mustAllocate(t, atlas, Size2(24, 32))
	d := mustAllocate(t, atlas, Size2(24, 8))

	if a.Rectangle.Min.X != 0 {
		t.Fatalf("a.min.x = %d, want 0", a.Rectangle.Min.X)
	}
	if b.Rectangle.Min.X != 32 {
		t.Fatalf("b.min.x = %d, want 32", b.Rectangle.Min.X)
	}
	if c.Rectangle.Min.X != 32 {
		t.Fatalf("c.min.x = %d, want 32", c.Rectangle.Min.X)
	}
	if d.Rectangle.Min.X != 0 {
		t.Fatalf("d.min.x = %d, want 0", d.Rectangle.Min.X)
	}
}

func TestBucketed_Vertical(t *testing.T) {
	options := DefaultOptions
	options.NumColumns = 2
	options.VerticalShelves = true
	atlas := BucketedAtlasAllocatorWithOptions(Size2(128, 256), options)

	if atlas.Size() != Size2(128, 256) {
		t.Fatalf("size = %v, want 128x256", atlas.Size())
	}

	a := mustAllocate(t, atlas, Size2(32, 16))
	b := mustAllocate(t, atlas, Size2(16, 32))

	if a.Rectangle.Size().Width < 32 || a.Rectangle.Size().Height < 16 {
		t.Fatalf("a = %v, too small", a.Rectangle)
	}
	if b.Rectangle.Size().Width < 16 || b.Rectangle.Size().Height < 32 {
		t.Fatalf("b = %v, too small", b.Rectangle)
	}

	c := mustAllocate(t, atlas, Size2(128, 128))

	atlas.Deallocate(a.ID)
	atlas.Deallocate(b.ID)
	atlas.Deallocate(c.ID)

	if !atlas.IsEmpty() {
		t.Fatal("atlas is not empty")
	}
	if atlas.AllocatedSpace() != 0 {
		t.Fatalf("allocated space = %d, want 0", atlas.AllocatedSpace())
	}
}

func TestBucketed_Clear(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(2048, 2048))

	sizes := []Size{
		Size2(8, 2), Size2(2, 8), Size2(16, 512), Size2(34, 34), Size2(34, 34), Size2(34, 34), Size2(34, 34),
		Size2(2, 8), Size2(2, 8), Size2(8, 2), Size2(2, 8), Size2(8, 2), Size2(8, 8), Size2(8, 8), Size2(8, 8),
		Size2(8, 8), Size2(82, 80), Size2(56, 56), Size2(64, 66), Size2(32, 32), Size2(32, 32), Size2(32, 32),
		Size2(32, 32), Size2(32, 32), Size2(32, 32), Size2(32, 32), Size2(32, 32), Size2(32, 32), Size2(40, 40),
		Size2(32, 32), Size2(256, 52), Size2(32, 32), Size2(256, 52), Size2(256, 52), Size2(256, 52),
		Size2(256, 52), Size2(256, 52), Size2(256, 52), Size2(155, 52), Size2(256, 52), Size2(32, 32),
		Size2(32, 32), Size2(32, 32), Size2(24, 24), Size2(64, 64), Size2(32, 32), Size2(84, 84), Size2(32, 32),
		Size2(8, 2), Size2(34, 34), Size2(34, 34), Size2(192, 192), Size2(192, 192), Size2(52, 52),
		Size2(144, 144), Size2(192, 192), Size2(32, 32), Size2(144, 144), Size2(24, 24), Size2(192, 192),
		Size2(192, 192), Size2(432, 243), Size2(32, 32), Size2(8, 2), Size2(2, 8), Size2(9, 9), Size2(14, 14),
		Size2(14, 14), Size2(14, 14), Size2(14, 14), Size2(8, 8), Size2(27, 27), Size2(27, 27), Size2(27, 27),
		Size2(27, 27), Size2(11, 12), Size2(29, 28), Size2(32, 32),
	}

	for range 500 {
		atlas.Clear()
		if atlas.AllocatedSpace() != 0 {
			t.Fatalf("allocated space after clear = %d, want 0", atlas.AllocatedSpace())
		}

		for _, s := range sizes {
			mustAllocate(t, atlas, s)
		}
	}
}

func TestBucketed_Fuzz01(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(1000, 1000))

	noAllocate(t, atlas, Size2(65280, 1))
	noAllocate(t, atlas, Size2(1, 65280))
}

func TestBucketed_Fuzz02(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(1000, 1000))

	noAllocate(t, atlas, Size2(255, 65599))
}

func TestBucketed_Fuzz03(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(1000, 1000))

	sizes := []Size{
		Size2(999, 128),
		Size2(168492810, 10),
		Size2(45, 96),
		Size2(-16711926, 0),
	}

	var allocations []Allocation
	var allocatedSpace int32

	for _, size := range sizes {
		if alloc, ok := atlas.Allocate(size); ok {
			allocations = append(allocations, alloc)
			allocatedSpace += alloc.Rectangle.Area()
			if allocatedSpace != atlas.AllocatedSpace() {
				t.Fatalf("allocated space = %d, want %d", atlas.AllocatedSpace(), allocatedSpace)
			}
		}
	}

	for _, alloc := range allocations {
		atlas.Deallocate(alloc.ID)

		allocatedSpace -= alloc.Rectangle.Area()
		if allocatedSpace != atlas.AllocatedSpace() {
			t.Fatalf("allocated space = %d, want %d", atlas.AllocatedSpace(), allocatedSpace)
		}
	}

	if atlas.AllocatedSpace() != 0 {
		t.Fatalf("allocated space = %d, want 0", atlas.AllocatedSpace())
	}
}

func TestBucketed_Fuzz04(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(1000, 1000))

	noAllocate(t, atlas, Size2(2560, 2147483647))
}

func TestBucketed_Fuzz05(t *testing.T) {
	atlas := NewBucketedAtlasAllocator(Size2(2048, 2048))

	noAllocate(t, atlas, Size2(0, -1978597547))
}
