package gpui

import (
	"encoding/binary"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"math/rand/v2"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"testing"
	"unicode"

	"github.com/megascan/gpui-go/scheduler"
)

type secInfiniteReader struct{}

func (secInfiniteReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 'A'
	}
	return len(p), nil
}

func TestSecurity_SEC012_ImageBodyBounded(t *testing.T) {
	if _, err := readImageBody(secInfiniteReader{}, 1<<20); !errors.Is(err, errImageAssetTooLarge) {
		t.Fatalf("infinite body: err = %v, want errImageAssetTooLarge", err)
	}
	data, err := readImageBody(io.LimitReader(secInfiniteReader{}, 1<<20), 1<<20)
	if err != nil || len(data) != 1<<20 {
		t.Fatalf("body at the limit: len=%d err=%v", len(data), err)
	}
}

type secHTTPClient struct {
	status   int
	body     string
	requests []*http.Request
}

func (c *secHTTPClient) Send(req *http.Request) *Task[*http.Response] {
	c.requests = append(c.requests, req)
	return scheduler.ReadyTask(&http.Response{
		StatusCode: c.status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader(c.body)),
		Request:    req,
	})
}

func (c *secHTTPClient) UserAgent() (string, bool) { return "sec", true }

func (c *secHTTPClient) Proxy() (*url.URL, bool) { return nil, false }

func secLoadURI(t *testing.T, client *secHTTPClient) scheduler.Result[*RenderImage] {
	t.Helper()
	var result scheduler.Result[*RenderImage]
	RunTest(t, func(tc *TestRun) {
		var load func(s *scheduler.Scope) scheduler.Result[*RenderImage]
		tc.Cx().Update(func(cx *App) {
			cx.SetHTTPClient(client)
			load = ImageAssetLoader{}.Load(ResourceURI(SharedURI("http://127.0.0.1:9/a.png")), cx)
		})
		tc.BlockTest(func(s *scheduler.Scope) {
			result = load(s)
		})
	})
	return result
}

func TestSecurity_SEC014_ImageErrorBodySanitized(t *testing.T) {
	client := &secHTTPClient{status: 500, body: "\x1b]0;owned\x07\x1b[2J\rforged‮ " + strings.Repeat("A", 1<<20) + "\nsecond\n"}
	result := secLoadURI(t, client)
	var cacheErr *ImageCacheError
	if !errors.As(result.Err, &cacheErr) || cacheErr.Kind != ImageCacheErrorKindBadStatus {
		t.Fatalf("err = %v, want a BadStatus ImageCacheError", result.Err)
	}
	if len(cacheErr.Body) > imageErrorBodyMaxBytes+len("…") {
		t.Errorf("error body is %d bytes", len(cacheErr.Body))
	}
	for _, r := range cacheErr.Body {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			t.Fatalf("error body keeps control character %U", r)
		}
	}
	if !strings.Contains(cacheErr.Body, "forged") {
		t.Errorf("error body lost its text: %q", cacheErr.Body)
	}
}

func TestSecurity_SEC015_ImageRequestHasDeadline(t *testing.T) {
	client := &secHTTPClient{status: 404}
	secLoadURI(t, client)
	if len(client.requests) != 1 {
		t.Fatalf("requests = %d", len(client.requests))
	}
	if _, ok := client.requests[0].Context().Deadline(); !ok {
		t.Fatal("image request has no deadline")
	}
}

func secChunkState(b []byte, chunks int) (uint64, uint64) {
	s0, s1 := fxSeed1, fxSeed2
	for k := range chunks {
		x := binary.LittleEndian.Uint64(b[16*k:])
		y := binary.LittleEndian.Uint64(b[16*k+8:])
		s0, s1 = s1, fxMultiplyMix(s0^x, fxPreventTrivialZeroCollapse^y)
	}
	return s0, s1
}

func secCollideBytes(target []byte, rng *rand.Rand) []byte {
	l := len(target)
	n := (l - 1) / 16
	tn := n
	t0, t1 := secChunkState(target, tn)
	t0 ^= binary.LittleEndian.Uint64(target[l-16:])
	t1 ^= binary.LittleEndian.Uint64(target[l-8:])
	b := make([]byte, l)
	for i := 16 * (n - 2); i < 16*(n-1); i++ {
		b[i] = byte(rng.Uint32())
	}
	_, s1 := secChunkState(b, n-1)
	binary.LittleEndian.PutUint64(b[l-16:], s1^t0)
	_, s1 = secChunkState(b, n)
	binary.LittleEndian.PutUint64(b[l-8:], s1^t1)
	return b
}

type secEchoAsset struct{}

func (secEchoAsset) Load(source []byte, _ *App) func(s *scheduler.Scope) string {
	return func(*scheduler.Scope) string { return string(source) }
}

func TestSecurity_SEC031_AssetKeyCollisionResistant(t *testing.T) {
	rng := rand.New(rand.NewPCG(31, 31))
	victim := []byte(strings.Repeat("https://cdn.trusted.example/avatar", 2) + "0123456789abcdef.png")
	if r := len(victim) % 16; r != 0 && r < 8 {
		t.Fatalf("victim length %d unsupported", len(victim))
	}
	forged := secCollideBytes(victim, rng)
	if Hash(victim) != Hash(forged) || string(victim) == string(forged) {
		t.Fatal("solver produced no FxHash collision")
	}
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *App) { cx.FetchAsset[secEchoAsset](forged) })
		cx.RunUntilParked()
		cx.Update(func(cx *App) { cx.FetchAsset[secEchoAsset](victim) })
		cx.RunUntilParked()
		got := cx.UpdateR(func(cx *App) string {
			v, _ := cx.FetchAsset[secEchoAsset](victim)
			return v
		})
		if got != string(victim) {
			t.Fatal("asset cache served the colliding source's result")
		}
	})
}

type secStringAsset struct{}

func (secStringAsset) Load(source string, _ *App) func(s *scheduler.Scope) string {
	return func(*scheduler.Scope) string { return source }
}

func TestSecurity_SEC036_AssetCacheBounded(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		cx := tc.Cx()
		for i := range maxAssetCacheEntries + 64 {
			cx.Update(func(cx *App) { cx.FetchAsset[secStringAsset](fmt.Sprintf("asset-%d", i)) })
			if i%512 == 0 {
				cx.RunUntilParked()
			}
		}
		cx.RunUntilParked()
		cx.Update(func(cx *App) {
			if n := len(cx.loadingAssets.entries); n > maxAssetCacheEntries {
				t.Errorf("asset cache holds %d entries, cap %d", n, maxAssetCacheEntries)
			}
			if cx.HasAsset[secStringAsset]("asset-0") {
				t.Error("least recently used asset was not evicted")
			}
			if !cx.HasAsset[secStringAsset](fmt.Sprintf("asset-%d", maxAssetCacheEntries+63)) {
				t.Error("most recent asset was evicted")
			}
		})
	})
}

func TestSecurity_SEC036_AssetCacheByteBudget(t *testing.T) {
	c := newAssetCache()
	released := 0
	for i := range 3 {
		slot := &assetSlot{key: loadingAssetKey{hash: uint64(i)}, load: secReleaser{released: &released}}
		c.insert(slot, nil)
		c.loaded(slot.key, maxAssetCacheBytes/2, nil)
	}
	if c.bytes > maxAssetCacheBytes || released != 1 || c.has(loadingAssetKey{hash: 0}) {
		t.Fatalf("bytes=%d released=%d", c.bytes, released)
	}
}

type secReleaser struct{ released *int }

func (r secReleaser) release() { *r.released++ }

func (r secReleaser) evict(*App) { *r.released++ }

func TestSecurity_SEC033_KeyContextParseNonIdentifier(t *testing.T) {
	for _, in := range []string{"(", "a .", "Editor mode=vim :", "a b=c )"} {
		if _, err := KeyContextParse(in); err == nil {
			t.Errorf("KeyContextParse(%q) succeeded", in)
		}
	}
	ctx, err := KeyContextParse("a=b =")
	if err != nil || !ctx.Contains("a") {
		t.Errorf("KeyContextParse(\"a=b =\") = %v, %v", ctx, err)
	}
}

func TestSecurity_SEC034_PredicateDepthLimited(t *testing.T) {
	for _, src := range []string{
		strings.Repeat("(", 1_000_000) + "a" + strings.Repeat(")", 1_000_000),
		strings.Repeat("!", 1_000_000) + "a",
		strings.Repeat("(", 1_000_000),
	} {
		if _, err := KeyBindingContextPredicateParse(src); !errors.Is(err, errPredicateTooDeep) {
			t.Errorf("deep predicate: err = %v", err)
		}
	}
	src := strings.Repeat("(", 64) + "a && !b" + strings.Repeat(")", 64)
	if _, err := KeyBindingContextPredicateParse(src); err != nil {
		t.Errorf("depth 64: %v", err)
	}
}

func TestSecurity_SEC032_TruncateWrappedLineInvalidUTF8(t *testing.T) {
	wrapper := buildWrapper()
	for _, line := range []string{"a\nW\xe6\x97", "日\xe6\x97", "abc \xe6\x97", "file\xe6\x97", "a\xff", "a b\xc3", "W\xe6"} {
		for lines := 1; lines <= 3; lines++ {
			for w := 1; w <= 120; w += 3 {
				for _, from := range []TruncateFrom{TruncateFromStart, TruncateFromEnd, TruncateFromMiddle} {
					func() {
						defer func() {
							if r := recover(); r != nil {
								t.Fatalf("TruncateWrappedLine(%q, %d, %d, %v) panicked: %v", line, w, lines, from, r)
							}
						}()
						runs := []TextRun{{Len: len(line)}}
						wrapper.TruncateWrappedLine(SharedString(line), Px(float32(w)), lines, "", runs, from)
						wrapper.TruncateWrappedLine(SharedString(line), Px(float32(w)), lines, "…", runs, from)
					}()
				}
			}
		}
	}
}

func secHeapInUse() uint64 {
	runtime.GC()
	runtime.GC()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.HeapInuse
}

func TestSecurity_SEC035_FontInternBoundedAndLigaRange(t *testing.T) {
	var f FontFeatures
	if err := json.Unmarshal([]byte(`{"liga":4294967296,"calt":1}`), &f); err != nil {
		t.Fatal(err)
	}
	for _, ft := range f.TagValueList() {
		if ft.Tag == "liga" {
			t.Errorf("out-of-range liga stored as %d", ft.Value)
		}
	}
	if enabled, ok := f.IsCaltEnabled(); !ok || !enabled {
		t.Error("calt lost")
	}
	a := NewFontFeatures([]FontFeature{{Tag: "liga", Value: 1}, {Tag: "calt", Value: 0}})
	b := NewFontFeatures([]FontFeature{{Tag: "liga", Value: 1}, {Tag: "calt", Value: 0}})
	if a != b || FontFallbacksFromFonts([]string{"A", "B"}) != FontFallbacksFromFonts([]string{"A", "B"}) {
		t.Fatal("equal lists are not equal")
	}
	if got := FontFallbacksFromFonts([]string{"A", "", "Bé"}).FallbackList(); len(got) != 3 || got[2] != "Bé" {
		t.Fatalf("FallbackList = %q", got)
	}
	const n = 100000
	before := secHeapInUse()
	for i := range n {
		NewFontFeatures([]FontFeature{{Tag: "liga", Value: uint32(i)}, {Tag: "calt", Value: uint32(bits.Reverse32(uint32(i)))}})
		FontFallbacksFromFonts([]string{fmt.Sprintf("Font %d with a moderately long family name to look like settings", i)})
	}
	after := secHeapInUse()
	if retained := int64(after) - int64(before); retained > 8<<20 {
		t.Errorf("intern tables retain %d bytes for %d dropped values", retained, n)
	}
}
