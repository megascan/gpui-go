//go:build securityharness && linux

package common

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

func TestSecB3_S05_ReadFdWithTimeoutTrickleHasNoTotalDeadline(t *testing.T) {
	var fds [2]int
	if err := unix.Pipe2(fds[:], unix.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	timeout := 100 * time.Millisecond
	trickle := 90 * time.Millisecond
	duration := 5 * time.Second
	go func() {
		defer unix.Close(fds[1])
		end := time.Now().Add(duration)
		for time.Now().Before(end) {
			if _, err := unix.Write(fds[1], []byte{'x'}); err != nil {
				return
			}
			time.Sleep(trickle)
		}
	}()
	start := time.Now()
	data, err := ReadFdWithTimeout(fds[0], timeout)
	blocked := time.Since(start)
	t.Logf("per-chunk timeout %v, peer wrote 1 byte every %v; ReadFdWithTimeout blocked %v, returned %d bytes, err=%v", timeout, trickle, blocked, len(data), err)
	if blocked > 10*timeout {
		t.Fatalf("FAIL: a trickling clipboard/DnD owner held ReadFdWithTimeout for %v (%.0fx the per-chunk timeout %v); production timeout is PipeReadTimeout=%v per chunk, so 1 byte every %v blocks the caller indefinitely (bounded only by MaxPipeReadSize=%d bytes)", blocked, float64(blocked)/float64(timeout), timeout, PipeReadTimeout, PipeReadTimeout-100*time.Millisecond, MaxPipeReadSize)
	}
}

func TestSecB3_S05ReadFdChild(t *testing.T) {
	if os.Getenv("B3_S05_CHILD") == "" {
		t.Skip("child only")
	}
	if os.Getenv("B3_S05_CHILD") == "control" {
		b3ControlAlloc()
	}
	var fds [2]int
	if err := unix.Pipe2(fds[:], unix.O_CLOEXEC); err != nil {
		t.Fatal(err)
	}
	go func() {
		defer unix.Close(fds[1])
		chunk := make([]byte, 1<<20)
		for range 300 {
			if _, err := unix.Write(fds[1], chunk); err != nil {
				return
			}
		}
	}()
	data, err := ReadFdWithTimeout(fds[0], 2*time.Second)
	var ms runtimeStats
	ms.read()
	fmt.Printf("RESULT len=%d err=%v peakRSSKiB=%d\n", len(data), err, ms.maxRSS)
	os.Exit(0)
}

type runtimeStats struct{ maxRSS int64 }

func (r *runtimeStats) read() {
	var ru syscall.Rusage
	_ = syscall.Getrusage(syscall.RUSAGE_SELF, &ru)
	r.maxRSS = ru.Maxrss
}

func TestSecB3_S05_ReadFdWithTimeoutOverCap(t *testing.T) {
	control := exec.Command("prlimit", b3MemCapArg(), "nice", "-n", "19", os.Args[0], "-test.run=^TestSecB3_S05ReadFdChild$", "-test.v")
	control.Env = append(os.Environ(), "B3_S05_CHILD=control", "GOMAXPROCS=2")
	if out, err := control.CombinedOutput(); err != nil || !bytes.Contains(out, []byte("control touched")) {
		t.Fatalf("harness-error: 512 MiB control child failed under the cap: %v\n%s", err, out)
	}
	cmd := exec.Command("prlimit", b3MemCapArg(), "nice", "-n", "19", os.Args[0], "-test.run=^TestSecB3_S05ReadFdChild$", "-test.v")
	cmd.Env = append(os.Environ(), "B3_S05_CHILD=1", "GOMAXPROCS=2")
	out, err := cmd.CombinedOutput()
	verdict := "clean"
	var exit *exec.ExitError
	switch {
	case bytes.Contains(out, []byte("out of memory")):
		verdict = "runtime-OOM"
	case errors.As(err, &exit):
		verdict = "exit " + strconv.Itoa(exit.ExitCode())
	}
	t.Logf("%s\n%s", verdict, out)
	if verdict != "clean" || !bytes.Contains(out, []byte("exceeds")) {
		t.Fatalf("FAIL: 300 MiB pipe did not end in a clean cap error: %s", verdict)
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
