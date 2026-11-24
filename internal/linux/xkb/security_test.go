//go:build linux

package xkb

import (
	"testing"

	"golang.org/x/sys/unix"
)

func TestSecurity_SEC024_keymap_fd_size(t *testing.T) {
	ctx := newTestContext(t)
	fd, err := unix.MemfdCreate("sec024-keymap", 0)
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("x")
	if _, err := unix.Write(fd, content); err != nil {
		unix.Close(fd)
		t.Fatal(err)
	}
	keymap, err := NewKeymapFromFd(ctx, fd, 3*unix.Getpagesize(), KeymapFormatTextV1, KeymapCompileNoFlags)
	if err == nil {
		t.Fatal("declared size larger than the fd must be rejected")
	}
	if keymap != nil {
		keymap.Close()
		t.Fatal("keymap must be nil for an oversized declaration")
	}
}
