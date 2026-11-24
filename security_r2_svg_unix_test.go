//go:build unix

package gpui

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestSecurity_SEC007_R2_05_SvgAssetRejectsSpecialFiles(t *testing.T) {
	load := func(path string) error {
		done := make(chan error, 1)
		go func() {
			done <- svgAsset{}.Load(SharedString(path), nil)(nil).Err
		}()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			t.Fatalf("loading %s blocked", path)
			return nil
		}
	}
	if err := load("/dev/zero"); err == nil {
		t.Fatal("/dev/zero loaded as an SVG")
	}
	fifo := filepath.Join(t.TempDir(), "pipe.svg")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := load(fifo); err == nil {
		t.Fatal("FIFO loaded as an SVG")
	}
	regular := filepath.Join(t.TempDir(), "icon.svg")
	if err := os.WriteFile(regular, []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := load(regular); err != nil {
		t.Fatalf("regular SVG file: %v", err)
	}
	big := filepath.Join(t.TempDir(), "big.svg")
	if err := os.WriteFile(big, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Truncate(big, imageAssetMaxBytes+1); err != nil {
		t.Fatal(err)
	}
	if err := load(big); !errors.Is(err, errImageAssetTooLarge) {
		t.Fatalf("oversized SVG file: want errImageAssetTooLarge, got %v", err)
	}
}
