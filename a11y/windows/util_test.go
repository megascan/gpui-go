//go:build windows

package windows

import (
	"slices"
	"testing"
	"unsafe"
)

func TestUtil_AcquireReusesDroppedAllocation(t *testing.T) {
	var scratch stringScratch
	buf := scratch.acquire()
	buf.buf = append(buf.buf, make([]uint16, 64)...)
	ptr := unsafe.SliceData(buf.buf)
	buf.release()
	buf = scratch.acquire()
	defer buf.release()
	if unsafe.SliceData(buf.buf) != ptr {
		t.Fatal("expected the dropped allocation to be reused")
	}
	if cap(buf.buf) < 64 {
		t.Fatalf("capacity %d < 64", cap(buf.buf))
	}
}

func TestUtil_SimultaneousLeasesAreDistinct(t *testing.T) {
	var scratch stringScratch
	a := scratch.acquire()
	defer a.release()
	b := scratch.acquire()
	defer b.release()
	a.buf = append(a.buf, 1)
	b.buf = append(b.buf, 2)
	if unsafe.SliceData(a.buf) == unsafe.SliceData(b.buf) {
		t.Fatal("leases share an allocation")
	}
	if a.buf[0] != 1 || b.buf[0] != 2 {
		t.Fatalf("got %d and %d", a.buf[0], b.buf[0])
	}
}

func TestUtil_OutsizedAllocationIsReleasedOnDrop(t *testing.T) {
	var scratch stringScratch
	buf := scratch.acquire()
	buf.buf = slices.Grow(buf.buf, maxRetainedScratchCapacity*4)
	if cap(buf.buf) <= maxRetainedScratchCapacity {
		t.Fatalf("capacity %d not above %d", cap(buf.buf), maxRetainedScratchCapacity)
	}
	buf.release()
	buf = scratch.acquire()
	defer buf.release()
	if cap(buf.buf) > maxRetainedScratchCapacity {
		t.Fatalf("capacity %d above %d", cap(buf.buf), maxRetainedScratchCapacity)
	}
}

func TestUtil_PoolRetainsAtMostMaxPooledBuffers(t *testing.T) {
	var scratch stringScratch
	bufs := make([]*stringBuffer, 0, maxPooledBuffers+1)
	for range maxPooledBuffers + 1 {
		buf := scratch.acquire()
		buf.buf = slices.Grow(buf.buf, 64)
		bufs = append(bufs, buf)
	}
	for _, buf := range bufs {
		buf.release()
	}
	reused := 0
	for range maxPooledBuffers + 1 {
		buf := scratch.acquire()
		defer buf.release()
		if cap(buf.buf) >= 64 {
			reused++
		}
	}
	if reused != maxPooledBuffers {
		t.Fatalf("reused %d buffers, want %d", reused, maxPooledBuffers)
	}
}

func TestUtil_AriaPropertiesSeparators(t *testing.T) {
	var buf []uint16
	w := newWideString(&buf)
	p := newAriaProperties(w)
	if p.hasProperties() {
		t.Fatal("empty writer reports properties")
	}
	p.writeBoolProperty("busy", true)
	p.writeUsizeProperty("level", 2)
	if got := string(utf16Decode(buf)); got != "busy=true;level=2" {
		t.Fatalf("got %q", got)
	}
}

func utf16Decode(u []uint16) []rune {
	out := make([]rune, 0, len(u))
	for _, c := range u {
		out = append(out, rune(c))
	}
	return out
}
