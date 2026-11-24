//go:build securityharness && linux

package xcursor

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func b3Cursor(ntoc uint32, index uint32, w, h uint32, pixels int) []byte {
	var b bytes.Buffer
	le := func(v uint32) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("Xcur")
	le(16)
	le(0x10000)
	le(ntoc)
	for range ntoc {
		le(0xfffd0002)
		le(24)
		le(index)
	}
	for b.Len() < int(index) {
		b.WriteByte(0)
	}
	le(0x24)
	le(0xfffd0002)
	le(24)
	le(1)
	le(w)
	le(h)
	le(0)
	le(0)
	le(0)
	b.Write(make([]byte, pixels))
	return b.Bytes()
}

func TestSecB3_S06aXcursorChild(t *testing.T) {
	name := os.Getenv("B3_S06A_XCUR_CASE")
	if name == "" {
		t.Skip("child only")
	}
	var data []byte
	switch name {
	case "control":
		b3ControlAlloc()
	case "max-dims-no-pixels":
		data = b3Cursor(1, 28, 0x7fff, 0x7fff, 0)
	case "toc-fanout-same-image":
		data = b3Cursor(4096, 4096*12+16, 512, 512, 512*512*4)
	}
	start := time.Now()
	imgs, err := ParseXcursorStream(bytes.NewReader(data))
	fmt.Printf("RESULT file=%dB images=%d err=%v elapsed=%v\n", len(data), len(imgs), err, time.Since(start))
	os.Exit(0)
}

func TestSecB3_S06a_XcursorAllocationFromHeader(t *testing.T) {
	var failures []string
	for _, name := range []string{"control", "max-dims-no-pixels", "toc-fanout-same-image"} {
		cmd := exec.Command("prlimit", b3MemCapArg(), "nice", "-n", "19", os.Args[0], "-test.run=^TestSecB3_S06aXcursorChild$", "-test.v")
		cmd.Env = append(os.Environ(), "B3_S06A_XCUR_CASE="+name, "GOMAXPROCS=2")
		out, err := cmd.CombinedOutput()
		verdict := "clean"
		var exit *exec.ExitError
		switch {
		case bytes.Contains(out, []byte("out of memory")):
			verdict = "runtime-OOM"
		case errors.As(err, &exit):
			if status, ok := exit.Sys().(syscall.WaitStatus); ok && status.Signaled() {
				verdict = "signal " + status.Signal().String()
			} else {
				verdict = "exit " + strconv.Itoa(exit.ExitCode())
			}
		}
		s := string(out)
		if len(s) > 500 {
			s = s[:500]
		}
		t.Logf("case %s: %s\n%s", name, verdict, s)
		if name == "control" && verdict != "clean" {
			t.Fatalf("harness-error: 512 MiB control child failed under the cap: %s", verdict)
		}
		if verdict != "clean" {
			failures = append(failures, name+"="+verdict)
		}
	}
	if len(failures) > 0 {
		t.Fatalf("FAIL: xcursor header-driven allocation under the baseline+3 GiB address-space cap: %v", failures)
	}
}

func TestSecB3_S06a_XcursorParserFuzz(t *testing.T) {
	seed := int64(0x5eedc0)
	t.Logf("seed %#x", seed)
	rng := rand.New(rand.NewSource(seed))
	base := b3Cursor(2, 40, 2, 2, 16)
	for i := 0; i < 20000; i++ {
		data := append([]byte(nil), base...)
		for range 1 + rng.Intn(4) {
			pos := rng.Intn(len(data))
			switch rng.Intn(3) {
			case 0:
				data[pos] = byte(rng.Intn(256))
			case 1:
				if pos&^3+4 <= len(data) {
					binary.LittleEndian.PutUint32(data[pos&^3:], rng.Uint32()%64)
				}
			case 2:
				data = data[:pos]
			}
			if len(data) < 4 {
				break
			}
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("FAIL: xcursor parser panicked on % x: %v", data, r)
				}
			}()
			_, _ = parseCursor(bytes.NewReader(data), 24)
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
