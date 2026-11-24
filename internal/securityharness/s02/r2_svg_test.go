//go:build securityharness

package s02

import (
	"encoding/base64"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func dataSVG(inner string) string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(inner))
}

func nestOnce(inner string, size int) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><image href="%s" width="%d" height="%d"/></svg>`, size, size, dataSVG(inner), size, size)
}

func TestR2_S02_01_NestedSvgLocalFileHref(t *testing.T) {
	secret := secretPNG(t, t.TempDir())
	inner := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><image href="%s" width="8" height="8"/></svg>`, secret)
	outer := nestOnce(nestOnce(inner, 8), 8)
	img, err := render(outer)
	if err != nil {
		t.Logf("render error (safe): %v", err)
		return
	}
	n := redPixels(img)
	t.Logf("nested data:svg -> data:svg -> <image href=%s>: red pixels=%d", secret, n)
	if n > 0 {
		t.Errorf("UNSAFE: nested SVG resolved a local file href (%d red pixels)", n)
	}
}

func TestR2_S02_02_NestedSvgDataPngBomb(t *testing.T) {
	var p string
	if !bombchild.IsChild(t) {
		bomb := bombPNG(30000, 30000)
		inner := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><image href="data:image/png;base64,%s" width="16" height="16"/></svg>`, base64.StdEncoding.EncodeToString(bomb))
		svg := nestOnce(nestOnce(inner, 16), 16)
		p = filepath.Join(t.TempDir(), "nested-bomb.svg")
		_ = os.WriteFile(p, []byte(svg), 0o600)
		t.Logf("svg %d bytes", len(svg))
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, p, func(p string) (string, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return renderSummary(string(b))
	})
	if parent {
		bombchild.ExpectSafe(t, o, "16x16 SVG -> data:svg -> data:svg -> data:PNG 30000x30000 zlib bomb")
	}
}

func nestedChain(depth, size int) string {
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="%d" height="%d" fill="red"/></svg>`, size, size, size, size)
	for range depth {
		svg = nestOnce(svg, size)
	}
	return svg
}

func memSummary(start time.Time, res string) string {
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return fmt.Sprintf("%s; TotalAlloc=%dMiB Sys=%dMiB elapsed=%s", res, ms.TotalAlloc>>20, ms.Sys>>20, time.Since(start).Round(time.Millisecond))
}

func TestR2_S02_03_NestedSvgDepthCanvasAmplification(t *testing.T) {
	const depth = 16
	svg := nestedChain(depth, 8192)
	if !bombchild.IsChild(t) {
		t.Logf("svg %d bytes, nesting depth %d, canvas 8192x8192", len(svg), depth)
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		start := time.Now()
		res, err := renderSummary(svg)
		if err != nil {
			return "", err
		}
		return memSummary(start, res), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("%d-byte SVG nesting data:svg images %d deep at 8192^2", len(svg), depth))
		if o.Elapsed > 20*time.Second {
			t.Errorf("UNSAFE CPU: nested SVG images took %s", o.Elapsed)
		}
	}
}

func TestR2_S02_04_NestedSvgUseFanoutCPU(t *testing.T) {
	const uses = 400
	inner := `<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="4" fill="red"/></svg>`
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="4096" height="4096"><defs><image id="i" href="%s" width="4" height="4"/></defs>`, dataSVG(inner))
	for range uses {
		b.WriteString(`<use xlink:href="#i"/>`)
	}
	b.WriteString(`</svg>`)
	svg := b.String()
	if !bombchild.IsChild(t) {
		t.Logf("svg %d bytes: %d <use> of one 4x4 data:svg image on a 4096^2 canvas", len(svg), uses)
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		start := time.Now()
		res, err := renderSummary(svg)
		if err != nil {
			return "", err
		}
		return memSummary(start, res), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "use fan-out of a nested SVG image")
		if o.Elapsed > 20*time.Second {
			t.Errorf("UNSAFE CPU: %d-byte SVG (%d uses of a 4x4 nested SVG image) took %s", len(svg), uses, o.Elapsed)
		}
	}
}

func dropShadowIcon(size int) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 24 24"><defs><filter id="s" x="-20%%" y="-20%%" width="140%%" height="140%%"><feGaussianBlur in="SourceAlpha" stdDeviation="0.6"/><feOffset dx="0" dy="0.5" result="o"/><feMerge><feMergeNode in="o"/><feMergeNode in="SourceGraphic"/></feMerge></filter></defs><circle cx="12" cy="12" r="9" fill="#3b82f6" filter="url(#s)"/></svg>`, size, size)
}

func TestR2_S02_05_LegitDropShadowRendersAtCommonSizes(t *testing.T) {
	var failed []string
	for _, size := range []int{256, 512, 1024, 1200, 1400, 1600, 2048} {
		start := time.Now()
		img, err := render(dropShadowIcon(size))
		el := time.Since(start).Round(time.Millisecond)
		if err != nil {
			t.Logf("size %4d: ERROR %v (%s)", size, err, el)
			failed = append(failed, fmt.Sprint(size))
			continue
		}
		s := img.Size(0)
		t.Logf("size %4d: rendered %dx%d in %s", size, s.Width, s.Height, el)
	}
	if len(failed) > 0 {
		t.Errorf("BEHAVIOUR-REGRESSION: a 24-unit drop-shadow icon (blur+offset+merge) is refused at %s px", strings.Join(failed, ", "))
	}
}

func TestR2_S02_06_LegitBlurredBackground1080p(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="1920" height="1080"><filter id="b"><feGaussianBlur stdDeviation="40"/></filter><g filter="url(#b)"><circle cx="400" cy="300" r="300" fill="#f0f"/><circle cx="1400" cy="700" r="350" fill="#0ff"/></g></svg>`
	start := time.Now()
	_, err := render(svg)
	t.Logf("1920x1080 blurred hero background: err=%v in %s", err, time.Since(start).Round(time.Millisecond))
	if err != nil {
		t.Errorf("BEHAVIOUR-REGRESSION: a 1920x1080 SVG with one blur filter is refused: %v", err)
	}
}

var fuzzSeeds = []string{
	`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64" viewBox="0 0 64 64"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1" gradientTransform="rotate(30)"><stop offset="0" stop-color="red"/><stop offset="1" stop-color="blue" stop-opacity="0.5"/></linearGradient><radialGradient id="r" cx="0.5" cy="0.5" r="0.5" fx="0.2" fy="0.2" spreadMethod="reflect"><stop offset="0.2" stop-color="#0f0"/><stop offset="1" stop-color="#00f"/></radialGradient><pattern id="p" width="8" height="8" patternUnits="userSpaceOnUse" patternTransform="skewX(20)"><rect width="4" height="4" fill="url(#g)"/></pattern><clipPath id="c" clipPathUnits="objectBoundingBox"><circle cx="0.5" cy="0.5" r="0.4"/></clipPath><mask id="m" maskUnits="userSpaceOnUse" x="0" y="0" width="64" height="64"><rect width="64" height="64" fill="white" opacity="0.5"/></mask><marker id="k" markerWidth="4" markerHeight="4" refX="2" refY="2" orient="auto-start-reverse"><path d="M0 0L4 2L0 4z"/></marker><symbol id="s" viewBox="0 0 10 10" preserveAspectRatio="xMidYMid slice"><rect width="10" height="10" rx="2" fill="url(#r)"/></symbol></defs><rect x="2" y="2" width="60" height="60" fill="url(#p)" stroke="url(#g)" stroke-width="3" stroke-dasharray="4 2" stroke-linejoin="miter" stroke-miterlimit="10" clip-path="url(#c)" mask="url(#m)"/><path d="M5 5 Q 20 40 40 5 T 60 60 A 10 15 30 1 0 30 50 C 10 10 50 10 5 60 Z" fill="none" stroke="black" marker-start="url(#k)" marker-mid="url(#k)" marker-end="url(#k)"/><use xlink:href="#s" x="10" y="10" width="20" height="20" transform="matrix(1 0.2 0.1 1 3 4)"/><polyline points="1,1 10,20 30,5 50,50" fill="none" stroke="#123" stroke-linecap="round"/><ellipse cx="30" cy="30" rx="10" ry="5" fill-rule="evenodd" opacity="0.7"/></svg>`,
	`<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><style>.a{fill:red;stroke:blue;stroke-width:2} #t{font-size:12px;font-family:sans-serif;font-weight:bold}</style><text id="t" x="4" y="20" class="a" textLength="40" lengthAdjust="spacingAndGlyphs" letter-spacing="1" word-spacing="2" dominant-baseline="middle" text-anchor="start">Hi <tspan dx="1 2 3" dy="-2" rotate="10 20" baseline-shift="super" font-style="italic">there</tspan></text><text><textPath href="#tp" startOffset="10%" method="stretch" side="right">along a path</textPath></text><path id="tp" d="M0 40 C 10 30 30 50 48 40"/><text writing-mode="tb" x="40" y="2" text-decoration="underline line-through">V</text></svg>`,
	`<svg xmlns="http://www.w3.org/2000/svg" width="40" height="40"><filter id="f" x="-10%" y="-10%" width="120%" height="120%" filterUnits="objectBoundingBox" primitiveUnits="userSpaceOnUse"><feTurbulence type="fractalNoise" baseFrequency="0.05 0.1" numOctaves="2" seed="3" stitchTiles="stitch" result="t"/><feColorMatrix type="hueRotate" values="90" in="t" result="c"/><feComponentTransfer in="c"><feFuncR type="table" tableValues="0 0.5 1"/><feFuncG type="gamma" amplitude="2" exponent="3" offset="0.1"/><feFuncB type="discrete" tableValues="0 1"/><feFuncA type="linear" slope="0.5" intercept="0.2"/></feComponentTransfer><feMorphology operator="dilate" radius="1 2"/><feConvolveMatrix order="3" kernelMatrix="0 1 0 1 -4 1 0 1 0" divisor="1" bias="0" targetX="1" targetY="1" edgeMode="wrap" preserveAlpha="true"/><feDiffuseLighting surfaceScale="2" diffuseConstant="1" lighting-color="white"><feDistantLight azimuth="45" elevation="30"/></feDiffuseLighting><feSpecularLighting specularExponent="10" specularConstant="1"><fePointLight x="10" y="10" z="20"/></feSpecularLighting><feComposite operator="arithmetic" k1="0.1" k2="0.5" k3="0.5" k4="0" in2="SourceGraphic"/><feDisplacementMap in2="t" scale="5" xChannelSelector="R" yChannelSelector="G"/><feTile/><feFlood flood-color="red" flood-opacity="0.3"/><feBlend mode="multiply" in2="SourceGraphic"/><feDropShadow dx="1" dy="1" stdDeviation="1"/><feImage href="#r2"/><feOffset dx="2" dy="3"/></filter><rect id="r2" x="5" y="5" width="30" height="30" fill="green" filter="url(#f)"/></svg>`,
}

var fuzzTokens = []string{"0", "-1", "1e38", "-1e38", "1e-38", "NaN", "inf", "-0", "4294967296", "99999999999999999999", "0.0000001", "%", "#", "url(#)", "url(#f)", "none", "", " ", "\x00", "<", ">", "\"", "&amp;", "&#0;", "&#x10FFFF;", "-", "e", "1e", ".", ",", "auto", "inherit", "65536", "2147483647", "-2147483648", "999%", "-100%"}

func mutateSVG(rng *rand.Rand, s string) string {
	b := []byte(s)
	for range 1 + rng.IntN(4) {
		if len(b) == 0 {
			break
		}
		i := rng.IntN(len(b))
		switch rng.IntN(6) {
		case 0:
			j := i
			for j < len(b) && (b[j] >= '0' && b[j] <= '9' || b[j] == '.' || b[j] == '-') {
				j++
			}
			tok := fuzzTokens[rng.IntN(len(fuzzTokens))]
			b = append(b[:i:i], append([]byte(tok), b[j:]...)...)
		case 1:
			j := min(len(b), i+1+rng.IntN(32))
			b = append(b[:i:i], b[j:]...)
		case 2:
			j := min(len(b), i+1+rng.IntN(64))
			k := rng.IntN(len(b))
			seg := append([]byte(nil), b[i:j]...)
			b = append(b[:k:k], append(seg, b[k:]...)...)
		case 3:
			b[i] = byte(rng.IntN(256))
		case 4:
			b = b[:i]
		case 5:
			tok := fuzzTokens[rng.IntN(len(fuzzTokens))]
			b = append(b[:i:i], append([]byte(tok), b[i:]...)...)
		}
	}
	return string(b)
}

func TestR2_S02_07_SvgMutationFuzzPanics(t *testing.T) {
	const fuzzSeed uint64 = 0x52325302
	t.Logf("seed=%#x", fuzzSeed)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 55*time.Second, "", func(string) (string, error) {
		debug.SetMaxStack(64 << 20)
		rng := rand.New(rand.NewPCG(fuzzSeed, 7))
		r := gpui.NewSvgRenderer(nil)
		deadline := time.Now().Add(35 * time.Second)
		iters := 0
		var found []string
		for time.Now().Before(deadline) && len(found) < 3 {
			src := fuzzSeeds[rng.IntN(len(fuzzSeeds))]
			for range 1 + rng.IntN(3) {
				src = mutateSVG(rng, src)
			}
			iters++
			done := make(chan struct{})
			go func() {
				defer close(done)
				defer func() {
					if p := recover(); p != nil {
						stack := string(debug.Stack())
						frames := []string{}
						for _, line := range strings.Split(stack, "\n") {
							if strings.Contains(line, "okage") || strings.Contains(line, "gpui-go/") && !strings.Contains(line, "securityharness") {
								frames = append(frames, strings.TrimSpace(line))
								if len(frames) >= 4 {
									break
								}
							}
						}
						found = append(found, fmt.Sprintf("panic %v at %s input=%q", p, strings.Join(frames, " <- "), src))
					}
				}()
				_, _ = r.RenderSingleFrame([]byte(src), 1.0)
			}()
			select {
			case <-done:
			case <-time.After(8 * time.Second):
				fmt.Printf("SLOW input still rendering after 8s: %q\n", src)
				return "", fmt.Errorf("slow input after %d iterations (render > 8s): %q", iters, src)
			}
		}
		if len(found) > 0 {
			fmt.Printf("PANIC %s\n", strings.Join(found, "\nPANIC "))
			return "", fmt.Errorf("%d panics in %d iterations; first: %s", len(found), iters, found[0])
		}
		return fmt.Sprintf("%d iterations, no panic", iters), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "SVG mutation fuzz")
		for _, line := range strings.Split(o.Output, "\n") {
			if strings.HasPrefix(line, "SLOW ") {
				t.Log(line)
				if dir := os.Getenv("SECHARNESS_EVIDENCE"); dir != "" {
					_ = os.WriteFile(filepath.Join(dir, "slow-input.txt"), []byte(line), 0o600)
				}
			}
		}
		if strings.Contains(o.Result, "slow input") {
			t.Errorf("CPU: a small mutated SVG renders for more than 8s: %s", o.Result)
		}
		if strings.Contains(o.Result, "panics in") {
			for _, line := range strings.Split(o.Output, "\n") {
				if strings.HasPrefix(line, "PANIC ") {
					t.Log(line)
				}
			}
			t.Errorf("PANIC reachable from ImageAssetLoader's SVG fallback: %s", o.Result)
		}
	}
}

func TestR2_S02_08_TextPathHugeCoordinateCPU(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text><textPath href="#tp">along a path</textPath></text><path id="tp" d="M1e38 40 C 10 30 30 50 48 40"/></svg>`
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 40*time.Second, "", func(string) (string, error) {
		start := time.Now()
		res, err := renderSummary(svg)
		if err != nil {
			return "", fmt.Errorf("%v after %s", err, time.Since(start).Round(time.Millisecond))
		}
		return res, nil
	})
	if parent {
		t.Logf("%d-byte SVG, textPath over a path starting at x=1e38: %s", len(svg), o)
		if !o.Safe() || o.Elapsed > 5*time.Second {
			t.Errorf("UNSAFE CPU: a %d-byte SVG with a textPath on a path with a 1e38 coordinate renders for %s (verdict %s)", len(svg), o.Elapsed.Round(time.Millisecond), o.Verdict)
		}
	}
}
