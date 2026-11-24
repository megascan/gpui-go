//go:build linux

package common

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	godbus "github.com/godbus/dbus/v5"
	"github.com/megascan/gpui-go"
	"golang.org/x/sys/unix"
)

func secStubLaunchers(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	bin := filepath.Join(root, "bin")
	calls := filepath.Join(root, "calls")
	for _, dir := range []string{bin, calls} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	script := fmt.Sprintf("#!/bin/sh\nf=$(mktemp %q/call.XXXXXXXX)\nprintf '%%s\\n' \"$@\" >\"$f\"\nmv \"$f\" \"$f.done\"\n", calls)
	for _, name := range []string{"xdg-open", "gio", "gnome-open", "kde-open", "kde-open5", "kde-open6", "dbus-launch"} {
		if err := os.WriteFile(filepath.Join(bin, name), []byte(script), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+":"+os.Getenv("PATH"))
	return calls
}

func secLauncherCalls(t *testing.T, dir string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, "call.*.done"))
	if err != nil {
		t.Fatal(err)
	}
	var calls []string
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			t.Fatal(err)
		}
		calls = append(calls, strings.TrimSpace(string(data)))
	}
	return calls
}

func secExecutor(t *testing.T) gpui.BackgroundExecutor {
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
	return c.BackgroundExecutor
}

func secFailingBus(calls *atomic.Int64) func() (*godbus.Conn, error) {
	return func() (*godbus.Conn, error) {
		calls.Add(1)
		return nil, errors.New("no session bus in tests")
	}
}

func TestSecurity_SEC016_RevealPathFIFODoesNotBlockWorker(t *testing.T) {
	calls := secStubLaunchers(t)
	executor := secExecutor(t)
	dir := t.TempDir()
	fifo := filepath.Join(dir, "dropped.fifo")
	if err := unix.Mkfifo(fifo, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if f, err := os.OpenFile(fifo, os.O_WRONLY|unix.O_NONBLOCK, 0); err == nil {
			_ = f.Close()
		}
	})
	var bus atomic.Int64
	RevealPathInternal(executor, secFailingBus(&bus), fifo, gpui.None[string](), gpui.GraphicalEnvironment{})
	deadline := time.Now().Add(3 * time.Second)
	for len(secLauncherCalls(t, calls)) == 0 && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	got := secLauncherCalls(t, calls)
	if bus.Load() == 0 || len(got) == 0 {
		t.Fatalf("RevealPath on a FIFO blocked its worker: bus calls=%d launches=%q", bus.Load(), got)
	}
	if got[0] != dir {
		t.Fatalf("revealed %q, want the FIFO's directory %q", got[0], dir)
	}
}

func TestSecurity_SEC018_OpenURIRefusesNonWebSchemes(t *testing.T) {
	calls := secStubLaunchers(t)
	executor := secExecutor(t)
	var bus atomic.Int64
	for _, uri := range []string{
		"file:///etc/passwd",
		"FILE:///etc/passwd",
		"smb://127.0.0.1/share/payload",
		"sftp://127.0.0.1/x",
		"javascript:alert(1)",
		"vscode://file/etc/passwd",
		"custom-handler:--flag=1",
	} {
		OpenURIInternal(executor, secFailingBus(&bus), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	allowed := []string{"https://example.invalid/a", "http://example.invalid/b", "mailto:someone@example.invalid"}
	for _, uri := range allowed {
		OpenURIInternal(executor, secFailingBus(&bus), uri, gpui.None[string](), gpui.GraphicalEnvironment{})
	}
	deadline := time.Now().Add(5 * time.Second)
	for len(secLauncherCalls(t, calls)) < len(allowed) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
	got := secLauncherCalls(t, calls)
	if len(got) != len(allowed) || bus.Load() != 0 {
		t.Fatalf("launches = %q, bus calls = %d; want exactly the %d web/mailto URLs", got, bus.Load(), len(allowed))
	}
	for _, launched := range got {
		if !strings.HasPrefix(launched, "http") && !strings.HasPrefix(launched, "mailto:") {
			t.Fatalf("non-web URL reached the launcher: %q", launched)
		}
	}
}

func TestSecurity_SEC027_ReadFdWithTimeoutHasTotalDeadline(t *testing.T) {
	read, write := newPipe(t)
	timeout := 200 * time.Millisecond
	stop := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer unix.Close(write)
		giveUp := time.After(3 * time.Second)
		for {
			select {
			case <-stop:
				return
			case <-giveUp:
				return
			case <-time.After(timeout / 4):
				if _, err := unix.Write(write, []byte{'x'}); err != nil {
					return
				}
			}
		}
	}()
	started := time.Now()
	_, err := ReadFdWithTimeout(read, timeout)
	elapsed := time.Since(started)
	close(stop)
	<-done
	if err == nil || !strings.Contains(err.Error(), "timed out") {
		t.Fatalf("trickling writer: err = %v", err)
	}
	if limit := timeout*PipeReadDeadlineFactor + time.Second; elapsed > limit {
		t.Fatalf("trickling writer held ReadFdWithTimeout for %v, want at most %v", elapsed, limit)
	}
}
