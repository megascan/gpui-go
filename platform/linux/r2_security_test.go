//go:build linux

package linux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSecurity_R2_11_OpenWithSystemRejectsURLs(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "-x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, target := range []string{"file:///etc/passwd", "smb://h/s", "javascript:alert(1)", "custom:--flag", "missing", ""} {
		if got, err := openWithSystemTarget(target); err == nil {
			t.Errorf("openWithSystemTarget(%q) = %q, want error", target, got)
		}
	}
	got, err := openWithSystemTarget("-x")
	if err != nil || got != filepath.Join(dir, "-x") {
		t.Fatalf("openWithSystemTarget(-x) = %q, %v", got, err)
	}
}
