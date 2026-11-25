//go:build securityharness

package s16

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
	"unicode"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/scheduler"
)

const s16ChildEnv = "S16_CHILD"

type s16Client struct {
	status   int
	body     func() io.ReadCloser
	requests []*http.Request
}

func (c *s16Client) Send(req *http.Request) *gpui.Task[*http.Response] {
	c.requests = append(c.requests, req)
	return scheduler.ReadyTask(&http.Response{
		Status:     fmt.Sprintf("%d %s", c.status, http.StatusText(c.status)),
		StatusCode: c.status,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     http.Header{},
		Body:       c.body(),
		Request:    req,
	})
}

func (c *s16Client) UserAgent() (string, bool) { return "s16", true }

func (c *s16Client) Proxy() (*url.URL, bool) { return nil, false }

type s16Infinite struct{}

func (s16Infinite) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'A'
	}
	return len(p), nil
}

func s16Load(t *testing.T, client *s16Client, uri string) scheduler.Result[*gpui.RenderImage] {
	t.Helper()
	var result scheduler.Result[*gpui.RenderImage]
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		var load func(s *scheduler.Scope) scheduler.Result[*gpui.RenderImage]
		tc.Cx().Update(func(cx *gpui.App) {
			cx.SetHTTPClient(client)
			load = gpui.ImageAssetLoader{}.Load(gpui.ResourceURI(gpui.SharedURI(uri)), cx)
		})
		tc.BlockTest(func(s *scheduler.Scope) {
			result = load(s)
		})
	})
	return result
}

func s16RunChild(t *testing.T, name string) {
	t.Helper()
	cmd := exec.Command("prlimit", "--stack=1048576", "--", "nice", "-n", "19", os.Args[0], "-test.run", "^"+name+"$", "-test.v", "-test.timeout", "50s")
	cmd.Env = append(os.Environ(), s16ChildEnv+"="+name, "GOMEMLIMIT=off")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	start := time.Now()
	err := cmd.Run()
	text := out.String()
	head := text
	if i := strings.Index(text, "goroutine "); i > 0 {
		head = text[:i]
	}
	if len(head) > 1500 {
		head = head[:1500]
	}
	t.Logf("child %s exited after %s: err=%v\n%s", name, time.Since(start).Round(time.Millisecond), err, head)
	switch {
	case strings.Contains(text, "S16_CLEAN_ERROR"):
		return
	case strings.Contains(text, "pthread_create failed"):
		t.Fatalf("%s: harness-error: child could not start threads under the cap", name)
	case strings.Contains(text, "out of memory"), strings.Contains(text, "cannot allocate memory"):
		t.Errorf("%s: process died with runtime OOM under an address-space cap of baseline+3 GiB (unbounded body read)", name)
	default:
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			t.Errorf("%s: child died: %v", name, err)
		} else {
			t.Errorf("%s: child finished without a clean bounded error", name)
		}
	}
}

func TestSecurity_S16_ImageLoaderInfiniteBodyBounded(t *testing.T) {
	if os.Getenv(s16ChildEnv) != "" {
		t.Skip("parent only")
	}
	s16RunChild(t, "TestS16Child_ImageLoaderInfiniteBody")
}

func TestS16Child_ImageLoaderInfiniteBody(t *testing.T) {
	if os.Getenv(s16ChildEnv) != t.Name() {
		t.Skip("child only")
	}
	s16CapSelf(3 << 30)
	client := &s16Client{status: 200, body: func() io.ReadCloser { return io.NopCloser(s16Infinite{}) }}
	result := s16Load(t, client, "http://127.0.0.1:9/infinite.png")
	if result.Err != nil {
		fmt.Printf("S16_CLEAN_ERROR %v\n", result.Err)
	}
}

func TestSecurity_S16_ImageLoaderErrorBodyEcho(t *testing.T) {
	firstLine := "\x1b]0;owned\x07\x1b[2J\rlevel=ERROR msg=\"forged entry\" " + strings.Repeat("A", 1<<20)
	client := &s16Client{status: 500, body: func() io.ReadCloser {
		return io.NopCloser(strings.NewReader(firstLine + "\nsecond line\n"))
	}}
	result := s16Load(t, client, "http://127.0.0.1:9/err.png")
	var cacheErr *gpui.ImageCacheError
	if !errors.As(result.Err, &cacheErr) {
		t.Fatalf("expected ImageCacheError, got %v", result.Err)
	}
	controls := 0
	for _, r := range cacheErr.Body {
		if unicode.IsControl(r) {
			controls++
		}
	}
	msg := cacheErr.Error()
	t.Logf("ImageCacheError.Body len=%d control chars=%d Error() len=%d prefix=%q", len(cacheErr.Body), controls, len(msg), msg[:min(len(msg), 120)])
	if len(cacheErr.Body) > 4096 {
		t.Errorf("error body echoed unbounded: %d bytes copied into ImageCacheError.Body and the logged error", len(cacheErr.Body))
	}
	if controls > 0 {
		t.Errorf("error body keeps %d terminal control characters (ESC, BEL, CR) that reach slog unescaped", controls)
	}
}

func TestSecurity_S16_ImageLoaderRequestCancellable(t *testing.T) {
	client := &s16Client{status: 404, body: func() io.ReadCloser { return io.NopCloser(strings.NewReader("")) }}
	s16Load(t, client, "http://127.0.0.1:9/a.png")
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d", len(client.requests))
	}
	ctx := client.requests[0].Context()
	_, hasDeadline := ctx.Deadline()
	t.Logf("request ctx done-chan=%v deadline=%v", ctx.Done() != nil, hasDeadline)
	if ctx.Done() == nil && !hasDeadline {
		t.Errorf("ImageAssetLoader sends requests with context.Background(): no deadline, and RemoveAsset/cancel cannot abort a slow body")
	}
}

func TestSecurity_S16_ImageLoaderSchemes(t *testing.T) {
	for _, uri := range []string{"file:///etc/passwd", "file://localhost/proc/self/environ", "gopher://127.0.0.1:70/x", "http://127.0.0.1:9/\r\nX-Injected: 1", "http://127.0.0.1:9/ x"} {
		client := &s16Client{status: 404, body: func() io.ReadCloser { return io.NopCloser(strings.NewReader("")) }}
		result := s16Load(t, client, uri)
		sent := ""
		if len(client.requests) > 0 {
			sent = client.requests[0].URL.String()
		}
		t.Logf("uri=%q handed to HTTPClient=%v sent=%q err=%v", uri, len(client.requests) > 0, sent, result.Err)
	}
}

func TestSecurity_S16_ControlCapAllowsBoundedWork(t *testing.T) {
	if os.Getenv(s16ChildEnv) != "" {
		t.Skip("parent only")
	}
	s16RunChild(t, "TestS16Child_Control")
}

func TestS16Child_Control(t *testing.T) {
	if os.Getenv(s16ChildEnv) != t.Name() {
		t.Skip("child only")
	}
	s16CapSelf(3 << 30)
	client := &s16Client{status: 500, body: func() io.ReadCloser { return io.NopCloser(io.LimitReader(s16Infinite{}, 512<<20)) }}
	result := s16Load(t, client, "http://127.0.0.1:9/bounded.png")
	if result.Err != nil {
		fmt.Printf("S16_CLEAN_ERROR control read 512 MiB then failed cleanly: %.80v\n", result.Err)
	}
}

func s16CapSelf(headroom uint64) uint64 {
	f, err := os.Open("/proc/self/status")
	if err != nil {
		panic(err)
	}
	defer f.Close()
	var vm uint64
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if rest, ok := strings.CutPrefix(sc.Text(), "VmSize:"); ok {
			kb, _ := strconv.ParseUint(strings.TrimSuffix(strings.TrimSpace(rest), " kB"), 10, 64)
			vm = kb << 10
		}
	}
	limit := vm + headroom
	if err := syscall.Setrlimit(syscall.RLIMIT_AS, &syscall.Rlimit{Cur: limit, Max: limit}); err != nil {
		panic(err)
	}
	fmt.Printf("S16_CAP baseline VmSize=%d MiB, RLIMIT_AS=%d MiB\n", vm>>20, limit>>20)
	return limit
}
