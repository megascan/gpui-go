//go:build linux || js

package wgpu

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/megascan/gpui-go/internal/webglsl"
)

func TestWgpuRendererWebgl_TranslatedShadersMatchEmbeddedWgsl(t *testing.T) {
	sum := sha256.Sum256([]byte(webglShaders))
	if got := hex.EncodeToString(sum[:]); got != webglsl.WebglWGSLSHA256 {
		t.Fatalf("webgl WGSL sha256 = %s, internal/webglsl was generated from %s; run go generate ./internal/webglsl", got, webglsl.WebglWGSLSHA256)
	}
}
