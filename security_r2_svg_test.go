package gpui

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	okagepath "github.com/xo/okage/path"
	"github.com/xo/okage/svg"
)

func securityDataSvg(inner string) string {
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(inner))
}

func securityNestedSvgChain(depth, size int) string {
	markup := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><rect width="%d" height="%d" fill="red"/></svg>`, size, size, size, size)
	for range depth {
		markup = fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d"><image href="%s" width="%d" height="%d"/></svg>`, size, size, securityDataSvg(markup), size, size)
	}
	return markup
}

func securitySvgRenderWork(t *testing.T, markup string, scaleFactor float32) error {
	t.Helper()
	parsed, err := NewSvgRenderer(nil).ParseSvg([]byte(markup))
	if err != nil {
		t.Fatal(err)
	}
	size := parsed.tree.Size()
	width, height := size.Width()*scaleFactor*SmoothSvgScaleFactor, size.Height()*scaleFactor*SmoothSvgScaleFactor
	scale := min(min(8192/width, 8192/height), 1)
	width, height = width*scale, height*scale
	return checkSvgRenderWork(parsed.tree, okagepath.TransformFromScale(width/size.Width(), height/size.Height()), f32AsU32(width), f32AsU32(height))
}

func TestSecurity_SEC005_R2_7_NestedSvgImagePixelBudget(t *testing.T) {
	renderer := NewSvgRenderer(nil)
	if _, err := renderer.RenderSingleFrame([]byte(securityNestedSvgChain(16, 8192)), 1); !errors.Is(err, errSvgNestedImageExceeded) {
		t.Fatalf("16 nested data:svg images at 8192^2: want errSvgNestedImageExceeded, got %v", err)
	}
	var fanout strings.Builder
	fmt.Fprintf(&fanout, `<svg xmlns="http://www.w3.org/2000/svg" width="4096" height="4096"><defs><image id="i" href="%s" width="4" height="4"/></defs>`, securityDataSvg(`<svg xmlns="http://www.w3.org/2000/svg" width="4" height="4"><rect width="4" height="4" fill="red"/></svg>`))
	for range 400 {
		fanout.WriteString(`<use href="#i"/>`)
	}
	fanout.WriteString(`</svg>`)
	if _, err := renderer.RenderSingleFrame([]byte(fanout.String()), 1); !errors.Is(err, errSvgNestedImageExceeded) {
		t.Fatalf("400 uses of a nested data:svg image at 4096^2: want errSvgNestedImageExceeded, got %v", err)
	}
	var icons strings.Builder
	icons.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="256" height="256">`)
	icon := securityDataSvg(`<svg xmlns="http://www.w3.org/2000/svg" width="32" height="32"><rect width="32" height="32" fill="red"/></svg>`)
	for i := range 8 {
		fmt.Fprintf(&icons, `<image href="%s" x="%d" y="0" width="32" height="32"/>`, icon, 32*i)
	}
	icons.WriteString(`</svg>`)
	if n := securitySvgRedPixels(t, icons.String()); n == 0 {
		t.Fatal("eight nested data:svg icons on a 256px SVG rendered no pixels")
	}
	if n := securitySvgRedPixels(t, securityNestedSvgChain(3, 256)); n == 0 {
		t.Fatal("three nested data:svg levels on a 256px SVG rendered no pixels")
	}
}

func TestSecurity_SEC006_R2_8_TextPathArclenBound(t *testing.T) {
	refused := map[string]string{
		"huge coordinate":   `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text><textPath href="#tp">along a path</textPath></text><path id="tp" d="M1e38 40 C 10 30 30 50 48 40"/></svg>`,
		"inline path":       `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text><textPath path="M1e38 40 C 10 30 30 50 48 40">along a path</textPath></text></svg>`,
		"text transform":    `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text transform="scale(1e20)"><textPath href="#tp">along a path</textPath></text><path id="tp" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
		"css transform":     `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><style>text{transform:scale(1e20)}</style><text><textPath href="#tp">along a path</textPath></text><path id="tp" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
		"viewBox scale":     `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48" viewBox="0 0 1e-20 1e-20"><text><textPath href="#tp">along a path</textPath></text><path id="tp" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
		"path transform":    `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text><textPath href="#tp">along a path</textPath></text><path id="tp" transform="scale(1e30)" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
		"circle radius":     `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text><textPath href="#tp">along a path</textPath></text><circle id="tp" r="1e38"/></svg>`,
		"use of text":       `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><text id="t"><textPath href="#tp">along a path</textPath></text><use href="#t" transform="scale(1e20)"/><path id="tp" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
		"hidden probe text": `<svg xmlns="http://www.w3.org/2000/svg" width="48" height="48"><style>tspan{display:none}</style><text transform="scale(1e20)"><textPath href="#tp">along a path</textPath></text><path id="tp" d="M0 40 C 10 30 30 50 48 40"/></svg>`,
	}
	renderer := NewSvgRenderer(nil)
	for name, markup := range refused {
		start := time.Now()
		_, err := renderer.RenderSingleFrame([]byte(markup), 1)
		if !errors.Is(err, errSvgTextPathTooLong) && !errors.Is(err, errSvgTextPathUnmeasured) {
			t.Errorf("%s: want a text path refusal, got %v", name, err)
		}
		if elapsed := time.Since(start); elapsed > 5*time.Second {
			t.Errorf("%s: refusal took %s", name, elapsed)
		}
	}
	nested := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><image href="%s" width="8" height="8"/></svg>`, securityDataSvg(refused["huge coordinate"]))
	start := time.Now()
	if _, err := renderer.RenderSingleFrame([]byte(nested), 1); err != nil {
		t.Fatalf("nested huge text path: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 5*time.Second {
		t.Fatalf("nested huge text path took %s", elapsed)
	}
	legit := `<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" viewBox="0 0 500 500" width="500" height="500"><defs><path id="curve" fill="none" d="M73.2,148.6c4-6.1,65.5-96.8,178.6-95.6c111.3,1.2,170.8,90.3,175.1,97"/></defs><text font-size="40" fill="red"><textPath xlink:href="#curve" startOffset="10%">Dangerous Curves Ahead</textPath></text></svg>`
	parsed, err := renderer.ParseSvg([]byte(legit))
	if err != nil {
		t.Fatalf("text on a curve: %v", err)
	}
	if !securitySvgHasText(parsed.tree.Root()) {
		t.Skip("no font available to lay out text")
	}
	if n := securitySvgRedPixels(t, legit); n == 0 {
		t.Fatal("text on a curve rendered no pixels")
	}
}

func TestSecurity_SEC006_BR_1_FilterLimitScalesToCommonSizes(t *testing.T) {
	dropShadow := func(size int) string {
		return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 24 24"><defs><filter id="s" x="-20%%" y="-20%%" width="140%%" height="140%%"><feGaussianBlur in="SourceAlpha" stdDeviation="0.6"/><feOffset dx="0" dy="0.5" result="o"/><feMerge><feMergeNode in="o"/><feMergeNode in="SourceGraphic"/></feMerge></filter></defs><circle cx="12" cy="12" r="9" fill="#3b82f6" filter="url(#s)"/></svg>`, size, size)
	}
	for _, size := range []int{256, 1024, 1600, 2048} {
		if err := securitySvgRenderWork(t, dropShadow(size), 1); err != nil {
			t.Errorf("drop-shadow icon at %dpx: %v", size, err)
		}
	}
	hero := `<svg xmlns="http://www.w3.org/2000/svg" width="1920" height="1080"><filter id="b"><feGaussianBlur stdDeviation="40"/></filter><g filter="url(#b)"><circle cx="400" cy="300" r="300" fill="#f0f"/><circle cx="1400" cy="700" r="350" fill="#0ff"/></g></svg>`
	if err := securitySvgRenderWork(t, hero, 1); err != nil {
		t.Errorf("1080p blurred background: %v", err)
	}
	blur8192 := `<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><filter id="f" x="0" y="0" width="1" height="1"><feGaussianBlur stdDeviation="400"/></filter><rect width="8192" height="8192" fill="red" filter="url(#f)"/></svg>`
	if err := securitySvgRenderWork(t, blur8192, 1); !errors.Is(err, errSvgFilterWorkExceeded) {
		t.Errorf("8192^2 blur: want errSvgFilterWorkExceeded, got %v", err)
	}
	offsetRegion := `<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><filter id="f" x="-2" y="-2" width="5" height="5"><feOffset dx="1"/></filter><rect width="8192" height="8192" fill="red" filter="url(#f)"/></svg>`
	if err := securitySvgRenderWork(t, offsetRegion, 1); !errors.Is(err, errSvgFilterWorkExceeded) {
		t.Errorf("one cheap filter over a 5x canvas region: want errSvgFilterWorkExceeded, got %v", err)
	}
}

func TestSecurity_SEC031_R2_04_SvgDataKeyUnforgeable(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"/>`)
	key, ok := NewSvg().Data(data).dataPath.Get()
	if !ok {
		t.Fatal("Data set no key")
	}
	if strings.HasPrefix(string(key), "__binary_svg__") || !strings.HasPrefix(string(key), svgKeyReservedPrefix) {
		t.Fatalf("data key %q is in the path key space", key)
	}
	if again, _ := NewSvg().Data(data).dataPath.Get(); again != key {
		t.Fatalf("same data gave keys %q and %q", key, again)
	}
	if other, _ := NewSvg().Data(append(data, ' ')).dataPath.Get(); other == key {
		t.Fatal("different data gave the same key")
	}
	if file := svgFileKey(key); file == key || !strings.HasPrefix(string(file), svgKeyReservedPrefix) {
		t.Fatalf("external path %q shares the data key space", file)
	}
	if svgFileKey("icons/a.svg") == "icons/a.svg" {
		t.Fatal("external path shares the asset path key space")
	}
}

func securitySvgHasText(group *svg.Group) bool {
	for _, child := range group.Children() {
		switch c := child.(type) {
		case *svg.Text:
			return true
		case *svg.Group:
			if securitySvgHasText(c) {
				return true
			}
		}
	}
	return false
}
