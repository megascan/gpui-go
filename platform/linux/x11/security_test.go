//go:build linux

package x11

import (
	"encoding/binary"
	"testing"
)

func TestSecurity_SEC029_incr_start_hint_capped(t *testing.T) {
	hint := make([]byte, 4)
	binary.LittleEndian.PutUint32(hint, 1<<30)
	if got := incrPreallocLen(32, hint); got != maxIncrPrealloc {
		t.Fatalf("INCR size hint of 1 GiB preallocates %d bytes, want %d", got, maxIncrPrealloc)
	}
	binary.LittleEndian.PutUint32(hint, 16)
	if got := incrPreallocLen(32, hint); got != 16 {
		t.Fatalf("small INCR hint = %d, want 16", got)
	}
	if got := incrPreallocLen(8, hint); got != 0 {
		t.Fatalf("non-32-bit INCR hint = %d, want 0", got)
	}
	if got := incrPreallocLen(32, hint[:2]); got != 0 {
		t.Fatalf("short INCR hint = %d, want 0", got)
	}
}
