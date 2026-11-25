//go:build securityharness

package s02

import (
	"bytes"
	"compress/flate"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/securityharness/lib/bombchild"
)

func svgzBomb(inflated int) []byte {
	const chunk = 1 << 20
	prefix := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><!--`)
	suffix := []byte(`--></svg>`)
	pad := bytes.Repeat([]byte{' '}, chunk)
	var deflated bytes.Buffer
	fw, _ := flate.NewWriter(&deflated, flate.BestCompression)
	_, _ = fw.Write(prefix)
	_, _ = fw.Write(pad)
	_ = fw.Flush()
	head := bytes.Clone(deflated.Bytes())
	deflated.Reset()
	_, _ = fw.Write(pad)
	_ = fw.Flush()
	body := bytes.Clone(deflated.Bytes())
	deflated.Reset()
	_, _ = fw.Write(suffix)
	_ = fw.Close()
	tail := deflated.Bytes()
	chunks := max((inflated-len(prefix)-len(suffix))/chunk, 2)
	crc := crc32.Update(0, crc32.IEEETable, prefix)
	for range chunks {
		crc = crc32.Update(crc, crc32.IEEETable, pad)
	}
	crc = crc32.Update(crc, crc32.IEEETable, suffix)
	out := bytes.NewBuffer([]byte{0x1f, 0x8b, 8, 0, 0, 0, 0, 0, 0, 0xff})
	out.Write(head)
	for range chunks - 1 {
		out.Write(body)
	}
	out.Write(tail)
	_ = binary.Write(out, binary.LittleEndian, crc)
	_ = binary.Write(out, binary.LittleEndian, uint32(len(prefix)+chunks*chunk+len(suffix)))
	return out.Bytes()
}

func svgzChild(t *testing.T, what string, build func([]byte) []byte) {
	var p string
	if !bombchild.IsChild(t) {
		bomb := build(svgzBomb(2 << 30))
		p = filepath.Join(t.TempDir(), "bomb.svgz")
		_ = os.WriteFile(p, bomb, 0o600)
		t.Logf("%s: %d bytes on disk, inflates to 2 GiB", what, len(bomb))
	}
	o, parent := bombchild.Run(t, bombchild.Baseline()+1<<30, childTimeout, p, func(p string) (string, error) {
		b, err := os.ReadFile(p)
		if err != nil {
			return "", err
		}
		start := time.Now()
		img, err := gpui.NewSvgRenderer(nil).RenderSingleFrame(b, 1.0)
		if err != nil {
			return "", err
		}
		s := img.Size(0)
		return memSummary(start, fmt.Sprintf("rendered %dx%d", s.Width, s.Height)), nil
	})
	if parent {
		bombchild.ExpectSafe(t, o, what)
		if o.Elapsed > 10*time.Second {
			t.Errorf("UNSAFE CPU: %s took %s", what, o.Elapsed)
		}
	}
}

func TestR3_S02_01_SvgzBomb(t *testing.T) {
	svgzChild(t, "2 MB top-level svgz", func(b []byte) []byte { return b })
}

func TestR3_S02_02_NestedSvgzBomb(t *testing.T) {
	svgzChild(t, "SVG embedding a 2 MB data:image/svg+xml svgz", func(b []byte) []byte {
		return []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="8" height="8"><image width="8" height="8" href="data:image/svg+xml;base64,` + base64.StdEncoding.EncodeToString(b) + `"/></svg>`)
	})
}

func TestR3_S02_03_SvgzControl(t *testing.T) {
	img, err := gpui.NewSvgRenderer(nil).RenderSingleFrame(svgzBomb(4<<20), 1.0)
	if err != nil {
		t.Fatalf("4 MiB svgz control: %v", err)
	}
	s := img.Size(0)
	t.Logf("4 MiB svgz control rendered %dx%d", s.Width, s.Height)
}
