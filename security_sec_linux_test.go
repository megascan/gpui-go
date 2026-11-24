package gpui

import (
	"errors"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/megascan/gpui-go/scheduler"
)

func secLoadPath(t *testing.T, path string) scheduler.Result[*RenderImage] {
	t.Helper()
	var load func(s *scheduler.Scope) scheduler.Result[*RenderImage]
	RunTest(t, func(tc *TestRun) {
		tc.Cx().Update(func(cx *App) {
			load = ImageAssetLoader{}.Load(ResourcePath(path), cx)
		})
	})
	done := make(chan scheduler.Result[*RenderImage], 1)
	go func() { done <- load(nil) }()
	select {
	case result := <-done:
		return result
	case <-time.After(10 * time.Second):
		t.Fatalf("loading %s did not return", path)
	}
	return scheduler.Result[*RenderImage]{}
}

func TestSecurity_SEC007_ImagePathRejectsSpecialFiles(t *testing.T) {
	fifo := filepath.Join(t.TempDir(), "fifo.png")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/dev/zero", fifo} {
		result := secLoadPath(t, path)
		var cacheErr *ImageCacheError
		if !errors.As(result.Err, &cacheErr) || cacheErr.Kind != ImageCacheErrorKindIo {
			t.Errorf("%s: err = %v, want an Io ImageCacheError", path, result.Err)
		}
	}
}
