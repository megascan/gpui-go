//go:build securityharness

package s02

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

const childTimeout = 50 * time.Second

func render(svg string) (*gpui.RenderImage, error) {
	return gpui.NewSvgRenderer(nil).RenderSingleFrame([]byte(svg), 1.0)
}

func renderSummary(svg string) (string, error) {
	start := time.Now()
	img, err := render(svg)
	if err != nil {
		return "", err
	}
	s := img.Size(0)
	return fmt.Sprintf("rendered %dx%d in %s", s.Width, s.Height, time.Since(start).Round(time.Millisecond)), nil
}

func redPixels(img *gpui.RenderImage) int {
	pix, ok := img.AsBytes(0)
	if !ok {
		return 0
	}
	n := 0
	for i := 0; i+4 <= len(pix); i += 4 {
		if pix[i+2] > 200 && pix[i+1] < 40 && pix[i] < 40 && pix[i+3] > 200 {
			n++
		}
	}
	return n
}

func secretPNG(t *testing.T, dir string) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := range 8 {
		for x := range 8 {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	p := filepath.Join(dir, "secret.png")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func imageSVG(href, attr string) string {
	return fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="8" height="8"><image %s="%s" x="0" y="0" width="8" height="8"/></svg>`, attr, href)
}

func TestS02_00_ControlInlineRed(t *testing.T) {
	img, err := render(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><rect width="8" height="8" fill="#ff0000"/></svg>`)
	if err != nil {
		t.Fatal(err)
	}
	if redPixels(img) == 0 {
		t.Fatalf("control: red rect not detected, pixel check is wrong")
	}
}

func TestS02_01_LocalFileImageAbsolute(t *testing.T) {
	p := secretPNG(t, t.TempDir())
	for _, attr := range []string{"href", "xlink:href"} {
		img, err := render(imageSVG(p, attr))
		if err != nil {
			t.Logf("%s: render error %v", attr, err)
			continue
		}
		if n := redPixels(img); n > 0 {
			t.Errorf("UNSAFE: SVG <image %s=%q> rendered %d pixels of a local file into the output", attr, p, n)
		}
	}
}

func TestS02_02_LocalFileImageRelative(t *testing.T) {
	var dir string
	if !bombchild.IsChild(t) {
		dir = t.TempDir()
		secretPNG(t, dir)
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, dir, func(dir string) (string, error) {
		if err := os.Chdir(dir); err != nil {
			return "", err
		}
		img, err := render(imageSVG("secret.png", "href"))
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("red=%d", redPixels(img)), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, "relative href")
		if strings.Contains(o.Result, "red=") && !strings.Contains(o.Result, "red=0") {
			t.Errorf("UNSAFE: relative <image href=\"secret.png\"> read a file from the process CWD: %s", o.Result)
		}
	}
}

func TestS02_03_FileURLAndProc(t *testing.T) {
	p := secretPNG(t, t.TempDir())
	for _, href := range []string{"file://" + p, "/proc/self/cwd/" + strings.TrimPrefix(p, "/")} {
		img, err := render(imageSVG(href, "href"))
		if err != nil {
			t.Logf("%s: %v", href, err)
			continue
		}
		if n := redPixels(img); n > 0 {
			t.Errorf("UNSAFE: <image href=%q> rendered %d local-file pixels", href, n)
		}
	}
}

func TestS02_04_DevZeroHref(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(imageSVG("/dev/zero", "href"))
	})
	if parent {
		bombchild.ExpectSafe(t, o, `<image href="/dev/zero">`)
	}
}

func TestS02_05_FifoHref(t *testing.T) {
	var p string
	if !bombchild.IsChild(t) {
		p = filepath.Join(t.TempDir(), "pipe.png")
		if err := syscall.Mkfifo(p, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 15*time.Second, p, func(p string) (string, error) {
		return renderSummary(imageSVG(p, "href"))
	})
	if parent {
		bombchild.ExpectSafe(t, o, `<image href=FIFO with no writer>`)
	}
}

func pngChunk(kind string, data []byte) []byte {
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, uint32(len(data)))
	b.WriteString(kind)
	b.Write(data)
	crc := crc32.NewIEEE()
	crc.Write([]byte(kind))
	crc.Write(data)
	_ = binary.Write(&b, binary.BigEndian, crc.Sum32())
	return b.Bytes()
}

func bombPNG(w, h int) []byte {
	var idat bytes.Buffer
	zw, _ := zlib.NewWriterLevel(&idat, zlib.BestCompression)
	row := make([]byte, w+1)
	for range h {
		_, _ = zw.Write(row)
	}
	_ = zw.Close()
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8] = 8
	b.Write(pngChunk("IHDR", ihdr))
	b.Write(pngChunk("IDAT", idat.Bytes()))
	b.Write(pngChunk("IEND", nil))
	return b.Bytes()
}

func TestS02_06_DataURIPngBomb(t *testing.T) {
	var p string
	if !bombchild.IsChild(t) {
		data := bombPNG(30000, 30000)
		svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><image href="data:image/png;base64,%s" width="16" height="16"/></svg>`, base64.StdEncoding.EncodeToString(data))
		p = filepath.Join(t.TempDir(), "bomb.svg")
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
		bombchild.ExpectSafe(t, o, "16x16 SVG embedding a data: PNG 30000x30000 zlib bomb")
	}
}

func useFanout(levels, fan int) string {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" width="64" height="64"><defs><rect id="g0" width="1" height="1" fill="red"/>`)
	for l := 1; l <= levels; l++ {
		fmt.Fprintf(&b, `<g id="g%d">`, l)
		for range fan {
			fmt.Fprintf(&b, `<use xlink:href="#g%d"/>`, l-1)
		}
		b.WriteString(`</g>`)
	}
	fmt.Fprintf(&b, `</defs><use xlink:href="#g%d"/></svg>`, levels)
	return b.String()
}

func TestS02_07_UseFanout(t *testing.T) {
	svg := useFanout(9, 10)
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(svg)
	})
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("%d-byte SVG with 9 levels x 10 <use> (10^9 nodes)", len(svg)))
	}
}

func TestS02_08a_Control8192Plain(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(`<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><rect width="8192" height="8192" fill="red"/></svg>`)
	})
	if parent && o.Verdict != "ok" {
		t.Fatalf("control: plain 8192^2 render does not fit the cap, filter test is invalid: %s", o)
	}
}

func TestS02_08b_SingleBlurAt8192(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(`<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><filter id="f" x="0" y="0" width="1" height="1"><feGaussianBlur stdDeviation="400"/></filter><rect width="8192" height="8192" fill="red" filter="url(#f)"/></svg>`)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "one 8192^2 gaussian blur")
		if o.Elapsed > 20*time.Second {
			t.Errorf("UNSAFE CPU: one blurred rect took %s", o.Elapsed)
		}
	}
}

func TestS02_08_BlurAt8192(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="8192" height="8192"><filter id="f" x="0" y="0" width="1" height="1"><feMorphology operator="dilate" radius="200"/><feGaussianBlur stdDeviation="400"/><feTurbulence baseFrequency="0.05" numOctaves="10"/></filter>`)
	for range 8 {
		b.WriteString(`<rect width="8192" height="8192" fill="red" filter="url(#f)"/>`)
	}
	b.WriteString(`</svg>`)
	svg := b.String()
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(svg)
	})
	if parent {
		bombchild.ExpectSafe(t, o, fmt.Sprintf("%d-byte SVG: 8 full-canvas 8192^2 morphology+blur+turbulence filters", len(svg)))
	}
}

func TestS02_09_EntityExpansion(t *testing.T) {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0"?><!DOCTYPE svg [<!ENTITY a0 "AAAAAAAAAA">`)
	for i := 1; i <= 9; i++ {
		fmt.Fprintf(&b, `<!ENTITY a%d "`, i)
		for range 10 {
			fmt.Fprintf(&b, "&a%d;", i-1)
		}
		b.WriteString(`">`)
	}
	b.WriteString(`]><svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><text>&a9;</text></svg>`)
	svg := b.String()
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(svg)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "billion-laughs DTD entities")
	}
}

func TestS02_10_DeepNesting(t *testing.T) {
	depth := 200000
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8">` + strings.Repeat("<g>", depth) + `<rect width="1" height="1"/>` + strings.Repeat("</g>", depth) + `</svg>`
	o, parent := bombchild.Run(t, bombchild.Baseline()+5<<30, childTimeout, "", func(string) (string, error) {
		return renderSummary(svg)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "200000 nested <g>")
	}
}

func TestS02_11_SelfReferencingFile(t *testing.T) {
	var p string
	if !bombchild.IsChild(t) {
		p = filepath.Join(t.TempDir(), "self.svg")
		_ = os.WriteFile(p, []byte(imageSVG(p, "href")), 0o600)
	}
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, 20*time.Second, p, func(p string) (string, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		return renderSummary(string(b))
	})
	if parent {
		bombchild.ExpectSafe(t, o, "SVG whose <image> references itself")
	}
}

func TestS02_12_HugeViewportClamp(t *testing.T) {
	o, parent := bombchild.Run(t, bombchild.DefaultLimit, childTimeout, "", func(string) (string, error) {
		return renderSummary(`<svg xmlns="http://www.w3.org/2000/svg" width="1e30" height="1"><rect width="10" height="1"/></svg>`)
	})
	if parent {
		bombchild.ExpectSafe(t, o, "width=1e30 height=1")
	}
}
