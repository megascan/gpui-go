//go:build securityharness && linux

package xrm

import (
	"bytes"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

const b3XrmChild = "B3_S06A_XRM_CASE"

func TestSecB3_S06aXrmChild(t *testing.T) {
	name := os.Getenv(b3XrmChild)
	if name == "" {
		t.Skip("child only")
	}
	dir := os.Getenv("B3_DIR")
	start := time.Now()
	var db *Database
	switch name {
	case "control":
		b3ControlAlloc()
	case "include-dev-zero":
		db, _ = NewFromGetPropertyReply(8, []byte("#include \"/dev/zero\"\nXft.dpi: 96\n"))
	case "include-fifo":
		db, _ = NewFromGetPropertyReply(8, []byte("#include \""+filepath.Join(dir, "fifo")+"\"\n"))
	case "include-fanout":
		db = NewFromDataWithBaseDirectory([]byte("#include \"a\"\n"), dir)
	case "include-self-depth":
		db = NewFromDataWithBaseDirectory([]byte("#include \"self\"\n"), dir)
	}
	fmt.Printf("RESULT entries=%d elapsed=%v\n", len(db.entries), time.Since(start))
	os.Exit(0)
}

func b3RunChild(t *testing.T, name, dir string, limit time.Duration) (string, string) {
	args := []string{b3MemCapArg(), "nice", "-n", "19", os.Args[0], "-test.run=^TestSecB3_S06aXrmChild$", "-test.v"}
	cmd := exec.Command("prlimit", args...)
	cmd.Env = append(os.Environ(), b3XrmChild+"="+name, "B3_DIR="+dir, "GOMAXPROCS=2")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	timedOut := false
	select {
	case err = <-done:
	case <-time.After(limit):
		timedOut = true
		_ = cmd.Process.Kill()
		err = <-done
	}
	verdict := "clean"
	var exit *exec.ExitError
	switch {
	case timedOut:
		verdict = "hang>" + limit.String()
	case bytes.Contains(out.Bytes(), []byte("out of memory")):
		verdict = "runtime-OOM"
	case errors.As(err, &exit):
		if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			verdict = "signal " + status.Signal().String()
		} else {
			verdict = "exit " + strconv.Itoa(exit.ExitCode())
		}
	}
	tail := out.String()
	if len(tail) > 600 {
		tail = tail[:600]
	}
	return verdict, tail
}

func TestSecB3_S06a_XrmIncludeFromRootProperty(t *testing.T) {
	dir, err := os.MkdirTemp("", "b3xrm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	if err := syscall.Mkfifo(filepath.Join(dir, "fifo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a"), []byte("#include \"a\"\n#include \"a\"\nk: v\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "self"), []byte("#include \"self\"\nk: v\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var failures []string
	for _, c := range []struct {
		name  string
		limit time.Duration
	}{{"control", 20 * time.Second}, {"include-self-depth", 20 * time.Second}, {"include-dev-zero", 30 * time.Second}, {"include-fifo", 5 * time.Second}, {"include-fanout", 10 * time.Second}} {
		verdict, out := b3RunChild(t, c.name, dir, c.limit)
		t.Logf("case %s: %s\n%s", c.name, verdict, out)
		if c.name == "control" && verdict != "clean" {
			t.Fatalf("harness-error: 512 MiB control child failed under the cap: %s", verdict)
		}
		if verdict != "clean" {
			failures = append(failures, c.name+"="+verdict)
		}
	}
	if len(failures) > 0 {
		t.Fatalf("FAIL: RESOURCE_MANAGER #include handling: %v", failures)
	}
}

func TestSecB3_S06a_XrmParserFuzz(t *testing.T) {
	seed := int64(0x5eed06a)
	t.Logf("seed %#x", seed)
	rng := rand.New(rand.NewSource(seed))
	alphabet := []byte("*.?!#:\\\n\t \"abcXft0123456789\x00\xff\xc3include")
	queries := [][2]string{{"Xft.dpi", "Xft.Dpi"}, {"a.b.c", "A.B.C"}, {"", ""}, {"..", ".."}, {"*", "*"}, {"a?b", "A?B"}}
	for i := 0; i < 20000; i++ {
		n := rng.Intn(200)
		data := make([]byte, n)
		for j := range data {
			data[j] = alphabet[rng.Intn(len(alphabet))]
		}
		data = bytes.ReplaceAll(data, []byte("#include"), []byte("#inklude"))
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("FAIL: xrm parser panicked on %q: %v", data, r)
				}
			}()
			db := NewFromData(data)
			for _, q := range queries {
				db.GetString(q[0], q[1])
				_, _, _ = db.GetUint32(q[0], q[1])
				_, _, _ = db.GetFloat32(q[0], q[1])
				_, _ = db.GetBool(q[0], q[1])
			}
		}()
	}
}

func b3MemCapArg() string {
	data, _ := os.ReadFile("/proc/self/status")
	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, "VmSize:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
					return "--as=" + strconv.FormatInt(kb*1024+3<<30, 10)
				}
			}
		}
	}
	return "--as=" + strconv.FormatInt(5<<30, 10)
}

func b3ControlAlloc() {
	buf := make([]byte, 512<<20)
	for i := 0; i < len(buf); i += 4096 {
		buf[i] = 1
	}
	fmt.Printf("RESULT control touched %d MiB\n", len(buf)>>20)
	os.Exit(0)
}
