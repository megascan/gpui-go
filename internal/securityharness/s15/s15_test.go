//go:build securityharness

package s15

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/bits"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
	"github.com/megascan/gpui-go/scheduler"
)

const (
	fxSeed1 uint64 = 0x243f6a8885a308d3
	fxSeed2 uint64 = 0x13198a2e03707344
	fxP     uint64 = 0xa4093822299f31d0
	seed    uint64 = 0x5515
)

const urlSafe = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"

func mix(x, y uint64) uint64 {
	hi, lo := bits.Mul64(x, y)
	return lo ^ hi
}

func chunkState(b []byte, chunks int) (uint64, uint64) {
	s0, s1 := fxSeed1, fxSeed2
	for k := range chunks {
		x := binary.LittleEndian.Uint64(b[16*k:])
		y := binary.LittleEndian.Uint64(b[16*k+8:])
		t := mix(s0^x, fxP^y)
		s0, s1 = s1, t
	}
	return s0, s1
}

func finalPair(b []byte) (uint64, uint64) {
	n := (len(b) - 1) / 16
	s0, s1 := chunkState(b, n)
	return s0 ^ binary.LittleEndian.Uint64(b[len(b)-16:]), s1 ^ binary.LittleEndian.Uint64(b[len(b)-8:])
}

func allURLSafe(v uint64) bool {
	for i := range 8 {
		if strings.IndexByte(urlSafe, byte(v>>(8*i))) < 0 {
			return false
		}
	}
	return true
}

func putRandom(rng *rand.Rand, b []byte) {
	for i := range b {
		b[i] = urlSafe[rng.IntN(len(urlSafe))]
	}
}

func collide(target, prefix string, rng *rand.Rand) (string, int, bool) {
	l := len(target)
	r := l % 16
	n := (l - 1) / 16
	if (r != 0 && r < 8) || n < 3 || len(prefix) > 16*(n-2) {
		return "", 0, false
	}
	t0, t1 := finalPair([]byte(target))
	b := []byte(strings.Repeat("a", l))
	copy(b, prefix)
	tries := 0
	for {
		tries++
		putRandom(rng, b[16*(n-2):16*(n-1)])
		_, s1 := chunkState(b, n-1)
		w0 := s1 ^ t0
		if allURLSafe(w0) {
			binary.LittleEndian.PutUint64(b[l-16:], w0)
			break
		}
	}
	freeEnd := min(16*n, l-16)
	for {
		tries++
		putRandom(rng, b[16*(n-1):freeEnd])
		_, s1 := chunkState(b, n)
		w1 := s1 ^ t1
		if allURLSafe(w1) {
			binary.LittleEndian.PutUint64(b[l-8:], w1)
			return string(b), tries, true
		}
	}
}

func collideBytes(target, attacker []byte, rng *rand.Rand) ([]byte, bool) {
	l := len(target)
	r := l % 16
	n := (l - 1) / 16
	if (r != 0 && r < 8) || n < 3 || len(attacker) > 16*(n-2) {
		return nil, false
	}
	t0, t1 := finalPair(target)
	b := make([]byte, l)
	copy(b, attacker)
	for i := 16 * (n - 2); i < 16*(n-1); i++ {
		b[i] = byte(rng.Uint32())
	}
	_, s1 := chunkState(b, n-1)
	binary.LittleEndian.PutUint64(b[l-16:], s1^t0)
	_, s1 = chunkState(b, n)
	binary.LittleEndian.PutUint64(b[l-8:], s1^t1)
	return b, true
}

func solidPNG(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	rng := rand.New(rand.NewPCG(uint64(w), uint64(h)))
	for y := range h {
		for x := range w {
			if c.R != 0 {
				c.G = uint8(rng.Uint32())
			}
			img.SetRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestS15_ImageIDCollision(t *testing.T) {
	t.Logf("seed=%#x", seed)
	rng := rand.New(rand.NewPCG(seed, 2))
	var victim []byte
	for w := 16; w < 64; w++ {
		victim = solidPNG(w, w, color.RGBA{R: 255, A: 255})
		if r := len(victim) % 16; r == 0 || r >= 8 {
			break
		}
	}
	attacker := solidPNG(1, 1, color.RGBA{B: 255, A: 255})
	forged, ok := collideBytes(victim, attacker, rng)
	if !ok {
		t.Fatalf("harness-error: victim length %d unsupported", len(victim))
	}
	a := gpui.ImageFromBytes(gpui.ImageFormatPng, victim)
	b := gpui.ImageFromBytes(gpui.ImageFormatPng, forged)
	decoded, err := png.Decode(bytes.NewReader(forged))
	if err != nil {
		t.Fatalf("harness-error: forged PNG does not decode: %v", err)
	}
	t.Logf("victim %d bytes ID=%#x; forged %d bytes ID=%#x; forged decodes to %v px, pixel(0,0)=%v", len(victim), a.ID, len(forged), b.ID, decoded.Bounds().Size(), decoded.At(0, 0))
	if a.ID == b.ID && !bytes.Equal(victim, forged) {
		t.Errorf("POISON: distinct image bytes share Image.ID (unkeyed FxHash of bytes); the decoded-image asset key (imageDecoderSource.Hash -> Image.ID) serves one for the other")
	}
}

type echoAsset struct{}

func (echoAsset) Load(source gpui.Resource, cx *gpui.App) func(s *scheduler.Scope) string {
	return func(*scheduler.Scope) string { return string(source.URI) }
}

const victim = "https://cdn.trusted.example/users/alice/avatars/profile-picture-128x128.png"

func TestS15_AssetCacheKeyCollision(t *testing.T) {
	t.Logf("seed=%#x", seed)
	rng := rand.New(rand.NewPCG(seed, 1))
	start := time.Now()
	evil, tries, ok := collide(victim, "https://evil.example/", rng)
	if !ok {
		t.Fatalf("harness-error: victim length %d unsupported by solver", len(victim))
	}
	ha := gpui.Hash(gpui.ResourceURI(gpui.SharedURI(victim)))
	hb := gpui.Hash(gpui.ResourceURI(gpui.SharedURI(evil)))
	t.Logf("victim=%s", victim)
	t.Logf("attacker=%s (found in %d tries, %s)", evil, tries, time.Since(start))
	t.Logf("Hash(victim)=%#x Hash(attacker)=%#x", ha, hb)
	if ha != hb || evil == victim {
		t.Fatalf("harness-error: solver produced no collision")
	}
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		cx.Update(func(cx *gpui.App) { cx.FetchAsset[echoAsset](gpui.ResourceURI(gpui.SharedURI(evil))) })
		cx.RunUntilParked()
		cx.Update(func(cx *gpui.App) { cx.FetchAsset[echoAsset](gpui.ResourceURI(gpui.SharedURI(victim))) })
		cx.RunUntilParked()
		got := cx.UpdateR(func(cx *gpui.App) string {
			v, _ := cx.FetchAsset[echoAsset](gpui.ResourceURI(gpui.SharedURI(victim)))
			return v
		})
		cx.RunUntilParked()
		hasImg := cx.UpdateR(func(cx *gpui.App) bool {
			cx.FetchAsset[gpui.ImgResourceLoader](gpui.ResourceURI(gpui.SharedURI(evil)))
			return cx.HasAsset[gpui.ImgResourceLoader](gpui.ResourceURI(gpui.SharedURI(victim)))
		})
		t.Logf("FetchAsset(victim) returned %q; HasAsset[ImgResourceLoader](victim) after loading only attacker = %v", got, hasImg)
		if got != string(victim) {
			t.Errorf("POISON: asset cache returned the attacker URI's result for the victim URI (key is a 64-bit unkeyed FxHash, source not compared)")
		}
	})
}

func TestS15_AssetCacheUnbounded(t *testing.T) {
	const n = 20000
	dir := t.TempDir()
	gpui.RunTest(t, func(tc *gpui.TestRun) {
		cx := tc.Cx()
		runtime.GC()
		var before runtime.MemStats
		runtime.ReadMemStats(&before)
		start := time.Now()
		for i := 0; i < n; i += 1000 {
			cx.Update(func(cx *gpui.App) {
				for j := i; j < i+1000; j++ {
					cx.FetchAsset[gpui.ImgResourceLoader](gpui.ResourcePath(filepath.Join(dir, fmt.Sprintf("missing-%07d-a-long-remote-looking-name.png", j))))
				}
			})
			cx.RunUntilParked()
		}
		runtime.GC()
		var after runtime.MemStats
		runtime.ReadMemStats(&after)
		first := cx.UpdateR(func(cx *gpui.App) bool {
			return cx.HasAsset[gpui.ImgResourceLoader](gpui.ResourcePath(filepath.Join(dir, fmt.Sprintf("missing-%07d-a-long-remote-looking-name.png", 0))))
		})
		grown := int64(after.HeapInuse) - int64(before.HeapInuse)
		t.Logf("%d distinct failed image loads in %s: heap +%d bytes (%.0f B/entry), first entry still cached=%v", n, time.Since(start), grown, float64(grown)/n, first)
		if first {
			t.Errorf("UNBOUNDED: every distinct image source stays in App.loadingAssets until RemoveAsset (no eviction; errors cached too)")
		}
	})
}

func TestS15_BackgroundPanicKillsProcess(t *testing.T) {
	o, parent := bombchild.Run(t, childLimit, 30*time.Second, "", func(string) (string, error) {
		d := gpui.NewThreadedDispatcher()
		ex := gpui.NewBackgroundExecutor(d)
		ex.Spawn(func(s *scheduler.Scope) (int, error) {
			var b []byte
			return int(b[len(os.Args)+3]), nil
		}).Detach()
		time.Sleep(2 * time.Second)
		return "process survived a panicking background task", nil
	})
	if parent {
		if !o.Safe() {
			t.Errorf("CRASH-AMPLIFIER: a panic inside a background task terminates the whole process: %s", o.Verdict)
		}
	}
}
