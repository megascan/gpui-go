//go:build securityharness

package bombchild

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"
)

const (
	envChild = "SECHARNESS_CHILD"
	envArg   = "SECHARNESS_ARG"
	sentinel = "SECHARNESS_RESULT:"
)

var DefaultLimit = Baseline() + 3<<30

func Baseline() int64 {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return 2 << 30
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "VmSize:"); ok {
			var kb int64
			if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d kB", &kb); err == nil {
				return kb << 10
			}
		}
	}
	return 2 << 30
}

type Outcome struct {
	Verdict string
	Result  string
	Output  string
	Elapsed time.Duration
}

func (o Outcome) Safe() bool {
	return o.Verdict == "error" || o.Verdict == "ok"
}

func (o Outcome) String() string {
	return fmt.Sprintf("verdict=%s elapsed=%s result=%q crash=%q", o.Verdict, o.Elapsed.Round(time.Millisecond), o.Result, o.Crash())
}

func (o Outcome) Crash() string {
	var parts []string
	lines := strings.Split(o.Output, "\n")
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "PANIC "), strings.HasPrefix(line, "fatal error:"), strings.HasPrefix(line, "panic:"), strings.HasPrefix(line, "runtime: out of memory"):
			parts = append(parts, tail(line, 300))
		case strings.Contains(line, "megascan/gpui-go") && !strings.Contains(line, "securityharness") && !strings.Contains(line, "testing.go") && i+1 < len(lines) && len(parts) < 8:
			parts = append(parts, strings.TrimSpace(line)+" @ "+strings.TrimSpace(lines[i+1]))
		}
	}
	return strings.Join(parts, " || ")
}

func IsChild(t *testing.T) bool {
	return os.Getenv(envChild) == t.Name()
}

func Arg() string {
	return os.Getenv(envArg)
}

func Run(t *testing.T, limit int64, timeout time.Duration, arg string, body func(arg string) (string, error)) (Outcome, bool) {
	t.Helper()
	if IsChild(t) {
		res, err := body(Arg())
		if err != nil {
			fmt.Printf("%serror %s\n", sentinel, oneLine(err.Error()))
		} else {
			fmt.Printf("%sok %s\n", sentinel, oneLine(res))
		}
		return Outcome{}, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	pattern := "^" + regexp.QuoteMeta(t.Name()) + "$"
	cmd := exec.CommandContext(ctx, "prlimit", fmt.Sprintf("--as=%d", limit), "--",
		"nice", "-n", "19", os.Args[0], "-test.run="+pattern, "-test.count=1", "-test.v",
		fmt.Sprintf("-test.timeout=%s", timeout+30*time.Second))
	cmd.Env = append(os.Environ(), envChild+"="+t.Name(), envArg+"="+arg, "GOTRACEBACK=single", "MALLOC_ARENA_MAX=2", "GOMAXPROCS=4")
	cmd.WaitDelay = 5 * time.Second
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	start := time.Now()
	err := cmd.Run()
	o := Outcome{Output: out.String(), Elapsed: time.Since(start)}
	text := o.Output
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, sentinel) {
			o.Result = strings.TrimPrefix(line, sentinel)
		}
	}
	switch {
	case ctx.Err() != nil:
		o.Verdict = "timeout"
	case err == nil && strings.HasPrefix(o.Result, "error"):
		o.Verdict = "error"
	case err == nil && strings.HasPrefix(o.Result, "panic"):
		o.Verdict = "panic"
	case err == nil && strings.HasPrefix(o.Result, "ok"):
		o.Verdict = "ok"
	case strings.Contains(text, "pthread_create failed"):
		o.Verdict = "harness-error"
	case strings.Contains(text, "out of memory"):
		o.Verdict = "oom"
	case strings.Contains(text, "panic:"):
		o.Verdict = "panic"
	case strings.Contains(text, "fatal error:"):
		o.Verdict = "fatal"
	case err != nil:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && !exitErr.Exited() {
			o.Verdict = "signal " + exitErr.String()
		} else {
			o.Verdict = "exit " + err.Error()
		}
	default:
		o.Verdict = "harness-no-sentinel"
	}
	if dir := os.Getenv("SECHARNESS_EVIDENCE"); dir != "" {
		_ = os.WriteFile(filepath.Join(dir, strings.ReplaceAll(t.Name(), "/", "_")+".log"), out.Bytes(), 0o600)
	}
	t.Logf("child %s (as cap %d MiB): %s", t.Name(), limit>>20, o)
	return o, true
}

func ExpectSafe(t *testing.T, o Outcome, what string) {
	t.Helper()
	switch {
	case o.Verdict == "harness-error":
		t.Errorf("HARNESS-ERROR %s: %s", what, o)
	case !o.Safe():
		t.Errorf("UNSAFE %s: %s", what, o)
	}
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " | ")
	if len(s) > 400 {
		s = s[:400] + "..."
	}
	return s
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
