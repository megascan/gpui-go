//go:build securityharness && linux

package linux

import "testing"

func TestSecB3_S05_PortalFileURIToPathDifferential(t *testing.T) {
	corpus := []string{
		"file:///a%00b", "file:///a%2F..%2F..%2Fetc%2Fpasswd", "file:///a/../../etc/passwd", "file://localhost/..",
		"file:///tmp/x%0Ab", "file://evil.example/share/x", "file://LOCALHOST/x", "FILE:///x", "file:/x", "file:x",
		"file:///", "file:///%E2%80%AEtxt.exe", "file://127.0.0.1/x", "file://user@localhost/x",
	}
	for _, in := range corpus {
		path, ok := fileURIToPath(in)
		t.Logf("%-40q -> %q ok=%v", in, path, ok)
	}
}
