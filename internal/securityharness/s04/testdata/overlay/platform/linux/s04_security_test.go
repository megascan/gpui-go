//go:build securityharness && linux

package linux

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
)

func s04LinuxStubs(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "bin")
	logDir := filepath.Join(root, "calls")
	for _, d := range []string{dir, logDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := fmt.Sprintf("#!/bin/sh\nf=$(mktemp %q/call.XXXXXXXX)\nprintf '%%s\\0' \"$(basename \"$0\")\" \"$@\" >\"$f.argv\"\nmv \"$f\" \"$f.done\"\nexit 0\n", logDir)
	for _, name := range []string{"xdg-open", "gio", "gnome-open", "kde-open", "kde-open5", "kde-open6", "dbus-launch", "dbus-send", "notify-send", "systemctl"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return logDir
}

func s04LinuxCalls(t *testing.T, logDir string, n int) [][]string {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		matches, _ := filepath.Glob(filepath.Join(logDir, "call.*.done"))
		if len(matches) >= n || time.Now().After(deadline) {
			sort.Strings(matches)
			var calls [][]string
			for _, done := range matches {
				argv, err := os.ReadFile(strings.TrimSuffix(done, ".done") + ".argv")
				if err != nil {
					t.Fatal(err)
				}
				calls = append(calls, strings.Split(strings.TrimSuffix(string(argv), "\x00"), "\x00"))
			}
			return calls
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestSecurity_S04_OpenWithSystemLeadingDash(t *testing.T) {
	logDir := s04LinuxStubs(t)
	dir := t.TempDir()
	t.Chdir(dir)
	targets := []string{"--help", "-x", "--manual", "--version"}
	for _, name := range targets {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("dropped file"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	p := newLinuxPlatform(gpui.WindowingModesHeadless)
	defer p.Close()
	for _, name := range targets {
		p.OpenWithSystem(name)
	}
	calls := s04LinuxCalls(t, logDir, len(targets))
	var unsafe []string
	for _, call := range calls {
		t.Logf("launched %q", call)
		if call[0] == "dbus-launch" {
			t.Errorf("platform construction reached dbus-launch")
		}
		if len(call) > 1 && strings.HasPrefix(call[len(call)-1], "-") {
			unsafe = append(unsafe, fmt.Sprintf("%s %q", call[0], call[1:]))
		}
	}
	if len(calls) != len(targets) {
		t.Errorf("expected %d launcher calls, got %d", len(targets), len(calls))
	}
	if len(unsafe) > 0 {
		t.Errorf("OpenWithSystem passes option-like paths to the launcher with no ./ prefix or --: %v", unsafe)
	}
}
