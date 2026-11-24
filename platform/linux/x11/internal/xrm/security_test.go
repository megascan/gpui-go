//go:build linux

package xrm

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func parseWithin(t *testing.T, data []byte, base string) *Database {
	t.Helper()
	done := make(chan *Database, 1)
	go func() { done <- NewFromDataWithBaseDirectory(data, base) }()
	select {
	case db := <-done:
		return db
	case <-time.After(10 * time.Second):
		t.Fatal("include expansion did not finish within 10s")
	}
	return nil
}

func TestSecurity_SEC025_xrm_include_guards(t *testing.T) {
	dir := t.TempDir()

	fifo := filepath.Join(dir, "fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	db := parseWithin(t, []byte("#include \"fifo\"\napp.ok: yes\n"), dir)
	if v, ok := db.GetString("app.ok", "App.Ok"); !ok || v != "yes" {
		t.Fatalf("entries after a FIFO include are lost: %q %v", v, ok)
	}

	big := filepath.Join(dir, "big")
	if err := os.WriteFile(big, append([]byte("app.big: found\n"), make([]byte, 2<<20)...), 0o600); err != nil {
		t.Fatal(err)
	}
	db = parseWithin(t, []byte("#include \"big\"\n"), dir)
	if _, ok := db.GetString("app.big", "App.Big"); ok {
		t.Fatal("include larger than the per-file cap was read")
	}

	self := filepath.Join(dir, "self")
	if err := os.WriteFile(self, []byte(strings.Repeat("#include \"self\"\n", 2)), 0o600); err != nil {
		t.Fatal(err)
	}
	parseWithin(t, []byte(strings.Repeat("#include \"self\"\n", 2)), dir)
}
