//go:build securityharness

package support

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
)

const s16SupportChildEnv = "S16_SUPPORT_CHILD"

func s16RunSupportChild(t *testing.T, name string) {
	t.Helper()
	cmd := exec.Command("prlimit", "--stack=1048576", "--", "nice", "-n", "19", os.Args[0], "-test.run", "^"+name+"$", "-test.v", "-test.timeout", "55s")
	cmd.Env = append(os.Environ(), s16SupportChildEnv+"="+name, "GOMEMLIMIT=off")
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
	case strings.Contains(text, "pthread_create failed"):
		t.Fatalf("%s: harness-error: child could not start threads under the cap", name)
	case strings.Contains(text, "out of memory"), strings.Contains(text, "cannot allocate memory"):
		t.Errorf("%s: process died with runtime OOM under an address-space cap of baseline+3 GiB (support client io.ReadAll of an endless body)", name)
	default:
		t.Errorf("%s: child ended without a clean bounded error: %v", name, err)
	}
}

func TestSecurity_S16_SupportClientInfiniteBodyBounded(t *testing.T) {
	if os.Getenv(s16SupportChildEnv) != "" {
		t.Skip("parent only")
	}
	s16RunSupportChild(t, "TestS16SupportChild_InfiniteChunked")
}

func TestS16SupportChild_InfiniteChunked(t *testing.T) {
	if os.Getenv(s16SupportChildEnv) != t.Name() {
		t.Skip("child only")
	}
	s16CapSelf(3 << 30)
	chunk := bytes.Repeat([]byte("A"), 1<<20)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		flusher, _ := w.(http.Flusher)
		for {
			if _, err := w.Write(chunk); err != nil {
				return
			}
			flusher.Flush()
		}
	}))
	defer server.Close()
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		executor := tc.BackgroundExecutor()
		executor.AllowParking()
		client, err := NewHTTPClientWithUserAgent("s16", executor)
		if err != nil {
			t.Fatal(err)
		}
		request, _ := http.NewRequest(http.MethodGet, server.URL+"/infinite.png", nil)
		task := client.Send(request)
		_ = task
		time.Sleep(30 * time.Second)
		fmt.Println("S16_CLEAN_ERROR survived 30s without OOM")
	})
}

func TestSecurity_S16_SupportClientRedirects(t *testing.T) {
	var internalHits atomic.Int64
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		internalHits.Add(1)
		w.WriteHeader(500)
		_, _ = io.WriteString(w, "internal-only admin page "+r.URL.Path+"\n")
	}))
	defer internal.Close()
	var loops atomic.Int64
	var edge *httptest.Server
	edge = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/to-file":
			http.Redirect(w, r, "file:///etc/passwd", http.StatusFound)
		case "/to-internal":
			http.Redirect(w, r, internal.URL+"/admin?token=1", http.StatusFound)
		case "/to-gopher":
			http.Redirect(w, r, "gopher://127.0.0.1:70/", http.StatusFound)
		case "/loop":
			loops.Add(1)
			http.Redirect(w, r, edge.URL+"/loop", http.StatusFound)
		default:
			w.WriteHeader(404)
		}
	}))
	defer edge.Close()
	client, err := NewHTTPClientWithUserAgent("s16", gpui.BackgroundExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/to-file", "/to-gopher", "/to-internal", "/loop"} {
		response, err := client.client.Get(edge.URL + path)
		status := 0
		var body string
		if err == nil {
			status = response.StatusCode
			b, _ := io.ReadAll(io.LimitReader(response.Body, 200))
			body = string(b)
			_ = response.Body.Close()
		}
		t.Logf("GET %s -> status=%d err=%v body=%q", path, status, err, body)
	}
	t.Logf("internal (different loopback port) hits=%d, redirect loop hops=%d", internalHits.Load(), loops.Load())
	if loops.Load() > 11 {
		t.Errorf("redirect loop not capped: %d", loops.Load())
	}
}

func TestSecurity_S16_SupportClientSlowBody(t *testing.T) {
	var sent atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)
		for range 600 {
			if _, err := w.Write([]byte("A")); err != nil {
				return
			}
			flusher.Flush()
			sent.Add(1)
			select {
			case <-r.Context().Done():
				return
			case <-time.After(250 * time.Millisecond):
			}
		}
	}))
	defer server.Close()
	client, err := NewHTTPClientWithUserAgent("s16", gpui.BackgroundExecutor{})
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		response, err := client.client.Get(server.URL + "/slow.png")
		if err == nil {
			_, err = io.ReadAll(response.Body)
			_ = response.Body.Close()
		}
		done <- err
	}()
	select {
	case err := <-done:
		t.Logf("slow body finished early: %v", err)
	case <-time.After(6 * time.Second):
		t.Logf("after 6s the 1-byte/250ms body is still being read (%d bytes); only the 60s whole-request Timeout bounds it (Timeout=%s)", sent.Load(), client.client.Timeout)
	}
	if client.client.Timeout <= 0 {
		t.Errorf("support client has no timeout")
	}
	server.CloseClientConnections()
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
