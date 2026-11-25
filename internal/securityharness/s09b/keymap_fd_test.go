//go:build securityharness && linux

package s09b

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"

	"github.com/megascan/gpui-go/internal/linux/xkb"
)

const b3ChildEnv = "B3_S09B_CASE"

func b3Memfd(t *testing.T, content []byte) int {
	fd, err := unix.MemfdCreate("b3-keymap", 0)
	if err != nil {
		t.Fatalf("harness: memfd: %v", err)
	}
	if _, err := unix.Write(fd, content); err != nil {
		t.Fatalf("harness: write: %v", err)
	}
	return fd
}

func TestSecB3_S09bChild(t *testing.T) {
	name := os.Getenv(b3ChildEnv)
	if name == "" {
		t.Skip("child only")
	}
	if err := xkb.Load(); err != nil {
		fmt.Println("HARNESS-ERROR load", err)
		os.Exit(3)
	}
	ctx := xkb.NewContext(xkb.ContextNoFlags)
	page := os.Getpagesize()
	var content []byte
	var size int
	switch name {
	case "exact-pages-declared-3x":
		content = bytes.Repeat([]byte{' '}, page)
		size = 3 * page
	case "comment-declared-3x":
		content = append([]byte("//"), bytes.Repeat([]byte{'x'}, page-2)...)
		size = 3 * page
	case "declared-1TiB":
		content = bytes.Repeat([]byte{' '}, page)
		size = 1 << 40
	case "size-1":
		content = []byte("x")
		size = 1
	case "size-0":
		content = nil
		size = 0
	case "empty-file-declared-page":
		content = nil
		size = page
	case "honest":
		content = append(bytes.Repeat([]byte{' '}, page-1), 0)
		size = page
	}
	fd := b3Memfd(t, content)
	keymap, err := xkb.NewKeymapFromFd(ctx, fd, size, xkb.KeymapFormatTextV1, xkb.KeymapCompileNoFlags)
	fmt.Printf("RESULT keymap=%v err=%v\n", keymap != nil, err)
	os.Exit(0)
}

func TestSecB3_S09b_NewKeymapFromFdDeclaredSizeVsFileSize(t *testing.T) {
	cases := []string{"honest", "size-0", "size-1", "empty-file-declared-page", "exact-pages-declared-3x", "comment-declared-3x", "declared-1TiB"}
	var failures []string
	for _, name := range cases {
		cmd := exec.Command("nice", "-n", "19", os.Args[0], "-test.run=^TestSecB3_S09bChild$", "-test.v")
		cmd.Env = append(os.Environ(), b3ChildEnv+"="+name)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		var err error
		select {
		case err = <-done:
		case <-time.After(20 * time.Second):
			_ = cmd.Process.Kill()
			err = <-done
		}
		verdict := "clean"
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				verdict = "signal " + status.Signal().String()
			} else {
				verdict = "exit " + strconv.Itoa(exit.ExitCode())
			}
		}
		if bytes.Contains(out.Bytes(), []byte("SIGBUS")) || bytes.Contains(out.Bytes(), []byte("unexpected fault address")) {
			verdict = "SIGBUS"
		}
		t.Logf("case %s: %s\n%s", name, verdict, lastLines(out.String(), 6))
		if verdict != "clean" {
			failures = append(failures, name+"="+verdict)
		}
	}
	if len(failures) > 0 {
		t.Fatalf("FAIL: NewKeymapFromFd crashed the process when the compositor-declared size exceeds the fd size: %v", failures)
	}
}

func lastLines(s string, n int) string {
	lines := bytes.Split([]byte(s), []byte("\n"))
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return string(bytes.Join(lines, []byte("\n")))
}
