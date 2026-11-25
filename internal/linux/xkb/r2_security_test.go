//go:build linux

package xkb

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestSecurity_R2_4_KeymapFdOversizeRejected(t *testing.T) {
	ctx := newTestContext(t)
	fd, err := unix.MemfdCreate("r2-4-keymap", 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := unix.Ftruncate(fd, MaxKeymapFdBytes+1); err != nil {
		unix.Close(fd)
		t.Fatal(err)
	}
	keymap, err := NewKeymapFromFd(ctx, fd, MaxKeymapFdBytes+1, KeymapFormatTextV1, KeymapCompileNoFlags)
	if err == nil || keymap != nil {
		t.Fatalf("keymap over the cap must be rejected, got %v %v", keymap, err)
	}
}
