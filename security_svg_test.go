package gpui

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/xo/okage/svg"
)

func securitySvgRedPixels(t *testing.T, markup string) int {
	t.Helper()
	img, err := NewSvgRenderer(nil).RenderSingleFrame([]byte(markup), 1)
	if err != nil {
		t.Fatal(err)
	}
	pix, _ := img.AsBytes(0)
	n := 0
	for i := 0; i+4 <= len(pix); i += 4 {
		if pix[i+2] > 200 && pix[i+1] < 40 && pix[i] < 40 && pix[i+3] > 200 {
			n++
		}
	}
	return n
}

func securityRedPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func securitySvgImage(href string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="8" height="8"><image href="%s" x="0" y="0" width="8" height="8"/></svg>`, href)
}

func securitySvgImageCount(group *svg.Group) int {
	n := 0
	for _, child := range group.Children() {
		switch c := child.(type) {
		case *svg.Image:
			n++
		case *svg.Group:
			n += securitySvgImageCount(c)
		}
	}
	return n
}

func TestSecurity_SEC004_SvgImageHrefLocalFiles(t *testing.T) {
	data := securityRedPNG(t)
	path := filepath.Join(t.TempDir(), "secret.png")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, href := range []string{path, "file://" + path, "/dev/zero"} {
		if n := securitySvgRedPixels(t, securitySvgImage(href)); n != 0 {
			t.Errorf("<image href=%q> rendered %d pixels of a local file", href, n)
		}
	}
	inline := "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)
	if n := securitySvgRedPixels(t, securitySvgImage(inline)); n == 0 {
		t.Errorf("data: PNG image was not rendered")
	}
}

func TestSecurity_SEC005_SvgDataImageBudget(t *testing.T) {
	bomb := securityPNGWithDims(30000, 30000)
	if err := reserveSvgRasterImage(newImageDecodeBudget(), svg.ImageKindPNG(bomb)); !errors.Is(err, errImageLimitsExceeded) {
		t.Fatalf("reserveSvgRasterImage: want errImageLimitsExceeded, got %v", err)
	}
	markup := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><image href="data:image/png;base64,%s" width="16" height="16"/></svg>`, base64.StdEncoding.EncodeToString(bomb))
	parsed, err := NewSvgRenderer(nil).ParseSvg([]byte(markup))
	if err != nil {
		t.Fatal(err)
	}
	if n := securitySvgImageCount(parsed.tree.Root()); n != 0 {
		t.Fatalf("data: PNG 30000x30000 kept as %d image nodes", n)
	}
}

func TestSecurity_SEC006_SvgFilterWorkLimit(t *testing.T) {
	_, err := NewSvgRenderer(nil).RenderSingleFrame([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><filter id="f" x="0" y="0" width="1" height="1"><feGaussianBlur stdDeviation="400"/></filter><rect width="8192" height="8192" fill="red" filter="url(#f)"/></svg>`), 1)
	if !errors.Is(err, errSvgFilterWorkExceeded) {
		t.Fatalf("8192^2 blur: want errSvgFilterWorkExceeded, got %v", err)
	}
	_, err = NewSvgRenderer(nil).RenderSingleFrame([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="64" height="64"><filter id="f"><feGaussianBlur stdDeviation="2"/><feMorphology radius="2"/></filter><mask id="m"><rect width="64" height="64" fill="white" filter="url(#f)"/></mask><rect width="64" height="64" fill="red" filter="url(#f)" mask="url(#m)"/></svg>`), 2)
	if err != nil {
		t.Fatalf("small filtered icon: %v", err)
	}
}
