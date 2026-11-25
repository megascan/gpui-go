//go:build securityharness && linux

package wayland

import (
	"strings"
	"testing"
)

var b3URLCorpus = []string{
	"file:///a%00b", "file:///a%2F..%2F..%2Fetc%2Fpasswd", "file:///a/../../etc/passwd", "file://localhost/..",
	"file:///tmp/x%0Ab", "file:///tmp/x%0D", "file://evil.example/share/x", "file://LOCALHOST/x", "FILE:///x",
	"file:/x", "file:x", "file:///", "file://", "file:///%E2%80%AEtxt.exe", "file:///a%", "file:///a b",
	"file:///-rf", "file://127.0.0.1/x", "file://[::1]/x", "file://user@localhost/x", "file:///x?q#f",
}

func TestSecB3_S05_WaylandFileURLToPathDifferential(t *testing.T) {
	var accepted []string
	for _, in := range b3URLCorpus {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("FAIL: fileURLToPath(%q) panicked: %v", in, r)
				}
			}()
			path, err := fileURLToPath(in)
			t.Logf("%-40q -> %q err=%v", in, path, err)
			if err == nil && (strings.ContainsAny(path, "\x00\n\r") || strings.Contains(path, "/../") || strings.HasSuffix(path, "/..")) {
				accepted = append(accepted, in+" => "+strings.ReplaceAll(path, "\x00", `\x00`))
			}
		}()
	}
	lines := rustLines("file:///a\r\nfile:///b\n\nfile:///c\r")
	t.Logf("rustLines -> %q", lines)
	if len(accepted) > 0 {
		t.Logf("INFO: paths with NUL/CR/LF or unnormalised '..' returned to the drop handler: %v", accepted)
	}
}
