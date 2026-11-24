//go:build securityharness && linux

package linux

import (
	"testing"

	"github.com/megascan/gpui-go"
)

func TestR2_S04_03_OpenWithSystemTakesURLs(t *testing.T) {
	logDir := s04LinuxStubs(t)
	t.Chdir(t.TempDir())
	p := newLinuxPlatform(gpui.WindowingModesHeadless)
	defer p.Close()
	targets := []string{"file:///etc/passwd", "smb://127.0.0.1/share/payload.desktop", "javascript:alert(1)", "s04custom:--flag"}
	for _, target := range targets {
		p.OpenWithSystem(target)
	}
	calls := s04LinuxCalls(t, logDir, len(targets))
	var reached []string
	for _, call := range calls {
		t.Logf("launched %q", call)
		reached = append(reached, call[len(call)-1])
	}
	if len(reached) > 0 {
		t.Errorf("OpenWithSystem hands URL-shaped strings to xdg-open, which opens them as URLs; OpenURL's scheme allowlist does not apply: %q", reached)
	}
}
