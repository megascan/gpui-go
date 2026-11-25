//go:build securityharness && linux

package common

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
	"golang.org/x/sys/unix"
)

type s04Call struct {
	program string
	args    []string
	env     []string
}

type s04Stubs struct {
	dir string
	log string
}

func s04InstallStubs(t *testing.T, exitCode int) *s04Stubs {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, "bin")
	logDir := filepath.Join(root, "calls")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	script := fmt.Sprintf("#!/bin/sh\nf=$(mktemp %q/call.XXXXXXXX)\nprintf '%%s\\0' \"$(basename \"$0\")\" \"$@\" >\"$f.argv\"\nenv -0 >\"$f.env\"\nmv \"$f\" \"$f.done\"\nexit %d\n", logDir, exitCode)
	for _, name := range []string{"xdg-open", "gio", "gnome-open", "kde-open", "kde-open5", "kde-open6", "dbus-launch", "dbus-send", "notify-send", "systemctl"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	return &s04Stubs{dir: dir, log: logDir}
}

func (s *s04Stubs) calls(t *testing.T) []s04Call {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(s.log, "call.*.done"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(matches)
	calls := make([]s04Call, 0, len(matches))
	for _, done := range matches {
		base := strings.TrimSuffix(done, ".done")
		argv, err := os.ReadFile(base + ".argv")
		if err != nil {
			t.Fatal(err)
		}
		env, err := os.ReadFile(base + ".env")
		if err != nil {
			t.Fatal(err)
		}
		fields := strings.Split(strings.TrimSuffix(string(argv), "\x00"), "\x00")
		envs := strings.Split(strings.TrimSuffix(string(env), "\x00"), "\x00")
		calls = append(calls, s04Call{program: fields[0], args: fields[1:], env: envs})
	}
	return calls
}

func (s *s04Stubs) waitCalls(t *testing.T, n int, timeout time.Duration) []s04Call {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		calls := s.calls(t)
		if len(calls) >= n || time.Now().After(deadline) {
			return calls
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func s04Common(t *testing.T) *LinuxCommon {
	t.Helper()
	eventLoop, err := NewEventLoop()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(eventLoop.Close)
	c, mainReceiver, powerReceiver := NewLinuxCommon(eventLoop.GetSignal())
	t.Cleanup(func() {
		c.Close()
		mainReceiver.Close()
		powerReceiver.Close()
	})
	return c
}

func s04FakeBus(counter *atomic.Int64) func() (*godbus.Conn, error) {
	return func() (*godbus.Conn, error) {
		counter.Add(1)
		return nil, errors.New("s04 fake bus: no session bus in sandbox")
	}
}

func TestSecurity_S04_OpenCommandsNeutraliseLeadingDash(t *testing.T) {
	for _, target := range []string{"--help", "-x", "--manual", "-\nfoo", "--version"} {
		for _, command := range openCommands(target) {
			last := command.Args[len(command.Args)-1]
			hasSep := slices.Contains(command.Args, "--")
			if strings.HasPrefix(last, "-") && !hasSep {
				t.Errorf("%s receives option-like target %q unprotected: %q", command.Args[0], target, command.Args)
			}
		}
	}
}

func TestSecurity_S04_OpenURIRejectsMalformed(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	for _, uri := range []string{" http://127.0.0.1/", "http://a b/", "-x", "--help", "relative/path", "", "\x00http://127.0.0.1/", "http://127.0.0.1/\n-x"} {
		OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	time.Sleep(500 * time.Millisecond)
	calls := stubs.calls(t)
	for _, call := range calls {
		t.Logf("launched %s %q", call.program, call.args)
		for _, arg := range call.args {
			if strings.HasPrefix(arg, "-") || strings.ContainsAny(arg, " \n\x00") {
				t.Errorf("malformed URI reached launcher: %s %q", call.program, call.args)
			}
		}
	}
	t.Logf("launcher calls=%d bus calls=%d", len(calls), busCalls.Load())
}

func TestSecurity_S04_OpenURIHasNoSchemeAllowlist(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	hostile := []string{
		"file://localhost/etc/passwd",
		"file://localhost/tmp/s04/evil.desktop",
		"smb://127.0.0.1/share/payload.exe",
		"sftp://127.0.0.1/x",
		"s04custom-handler:--flag=1",
		"javascript:alert(1)",
		"ms-msdt:/id",
		"vscode://file/etc/passwd",
	}
	for _, uri := range hostile {
		OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	calls := stubs.waitCalls(t, len(hostile), 10*time.Second)
	launched := map[string]bool{}
	for _, call := range calls {
		t.Logf("launched %s %q", call.program, call.args)
		launched[call.args[len(call.args)-1]] = true
	}
	var reached []string
	for _, uri := range hostile {
		if launched[uri] {
			reached = append(reached, uri)
		}
	}
	if len(reached) > 0 {
		t.Errorf("OpenURIInternal hands non-web schemes to the desktop launcher (no allowlist): %q", reached)
	}
}

func TestSecurity_S04_ActivationTokenCannotInjectEnv(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	token := "s04tok\nLD_PRELOAD=/tmp/s04-evil.so\nXDG_S04_INJECTED=1"
	OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), "https://127.0.0.1/a", gpui.Some(token), gpui.GraphicalEnvironment{})
	calls := stubs.waitCalls(t, 1, 10*time.Second)
	if len(calls) == 0 {
		t.Fatal("no launcher call observed")
	}
	tokens := 0
	for _, kv := range calls[0].env {
		name, value, _ := strings.Cut(kv, "=")
		switch name {
		case "XDG_ACTIVATION_TOKEN":
			tokens++
			if value != token {
				t.Errorf("token mangled: %q", value)
			}
		case "LD_PRELOAD", "XDG_S04_INJECTED":
			t.Errorf("token smuggled a separate env var: %q", kv)
		}
	}
	if tokens != 1 {
		t.Errorf("XDG_ACTIVATION_TOKEN entries = %d", tokens)
	}
}

func TestSecurity_S04_ActivationTokenNULFallsBackToPortal(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	var busCalls atomic.Int64
	OpenURIInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), "https://127.0.0.1/a", gpui.Some("tok\x00en"), gpui.GraphicalEnvironment{})
	deadline := time.Now().Add(5 * time.Second)
	for busCalls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	t.Logf("launcher calls=%d bus calls=%d", len(stubs.calls(t)), busCalls.Load())
	if len(stubs.calls(t)) != 0 {
		t.Errorf("launcher ran with NUL in env")
	}
}

func TestSecurity_S04_RevealPathLeadingDashDirectory(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, "-x"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "--help", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "--help", "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	var busCalls atomic.Int64
	for _, p := range []string{"-x", "--help", "--help/f"} {
		RevealPathInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), p, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	calls := stubs.waitCalls(t, 3, 10*time.Second)
	for _, call := range calls {
		t.Logf("launched %s %q", call.program, call.args)
		last := call.args[len(call.args)-1]
		if strings.HasPrefix(last, "-") && !slices.Contains(call.args, "--") {
			t.Errorf("RevealPath passed option-like dir: %s %q", call.program, call.args)
		}
	}
}

func TestSecurity_S04_RevealPathFIFOStarvesBackgroundExecutor(t *testing.T) {
	s04InstallStubs(t, 0)
	c := s04Common(t)
	workers := max(runtime.NumCPU(), minThreads)
	dir := t.TempDir()
	fifos := make([]string, workers)
	for i := range fifos {
		fifos[i] = filepath.Join(dir, fmt.Sprintf("dropped-%02d.fifo", i))
		if err := unix.Mkfifo(fifos[i], 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var busCalls atomic.Int64
	for _, fifo := range fifos {
		RevealPathInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), fifo, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	time.Sleep(300 * time.Millisecond)
	var ran atomic.Bool
	c.BackgroundExecutor.Spawn(func(s *scheduler.Scope) (struct{}, error) {
		ran.Store(true)
		return struct{}{}, nil
	}).Detach()
	deadline := time.Now().Add(5 * time.Second)
	for !ran.Load() && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	starved := !ran.Load()
	for _, fifo := range fifos {
		done := make(chan struct{})
		go func() {
			defer close(done)
			if f, err := os.OpenFile(fifo, os.O_WRONLY|unix.O_NONBLOCK, 0); err == nil {
				_ = f.Close()
			}
		}()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
		}
	}
	unblockDeadline := time.Now().Add(5 * time.Second)
	for !ran.Load() && time.Now().Before(unblockDeadline) {
		time.Sleep(20 * time.Millisecond)
	}
	t.Logf("workers=%d fifos=%d sentinel ran before unblock=%v after unblock=%v", workers, len(fifos), !starved, ran.Load())
	if starved {
		t.Errorf("RevealPath on %d FIFOs blocked every background worker: an unrelated background task did not run within 5s", workers)
	}
}

func TestSecurity_S04_RevealPathSingleFIFOHangsForever(t *testing.T) {
	stubs := s04InstallStubs(t, 0)
	c := s04Common(t)
	fifo := filepath.Join(t.TempDir(), "x.fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	var busCalls atomic.Int64
	RevealPathInternal(c.BackgroundExecutor, s04FakeBus(&busCalls), fifo, gpui.None[string](), gpui.GraphicalEnvironment{})
	time.Sleep(3 * time.Second)
	hung := busCalls.Load() == 0 && len(stubs.calls(t)) == 0
	if f, err := os.OpenFile(fifo, os.O_WRONLY|unix.O_NONBLOCK, 0); err == nil {
		_ = f.Close()
	}
	calls := stubs.waitCalls(t, 1, 5*time.Second)
	var buf bytes.Buffer
	for _, call := range calls {
		fmt.Fprintf(&buf, "%s %q; ", call.program, call.args)
	}
	t.Logf("after writer opened: bus=%d launches=%s", busCalls.Load(), buf.String())
	if hung {
		t.Errorf("RevealPath on a FIFO blocked its worker for 3s+ in os.Open (no O_NONBLOCK / type check)")
	}
}
