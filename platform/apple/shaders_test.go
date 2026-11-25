//go:build darwin

package apple

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

const shadersMetalSHA256 = "bb86d60f6e04c4ecf0994da41a9dfa4eb169679c6c93482b08fc956a28c292ec"

func TestShaders_MetalSourceMatchesUpstream(t *testing.T) {
	sum := sha256.Sum256([]byte(shadersMetal))
	if got := hex.EncodeToString(sum[:]); got != shadersMetalSHA256 {
		t.Fatalf("sha256(shaders.metal) = %s, want %s", got, shadersMetalSHA256)
	}
	upstream, err := os.ReadFile("../../_ref/gpui_apple/src/shaders.metal")
	if err != nil {
		t.Logf("_ref not available, checked the pinned hash only: %v", err)
		return
	}
	if string(upstream) != shadersMetal {
		t.Fatal("shaders.metal differs from _ref/gpui_apple/src/shaders.metal")
	}
}

func TestShaders_StitchedSourceLayout(t *testing.T) {
	src := stitchedShaderSource()
	if !strings.HasPrefix(src, "#ifndef SCENE_H\n") {
		t.Fatal("stitched source does not start with the scene.h include guard")
	}
	if !strings.HasSuffix(src, shadersMetal) || !strings.Contains(src, "#endif  /* SCENE_H */\n\n") {
		t.Fatal("stitched source is not scene.h + \"\\n\" + shaders.metal")
	}
}

func TestShaders_CompileAndResolveFunctions(t *testing.T) {
	if err := mtl.Load(); err != nil {
		t.Skipf("Metal unavailable: %v", err)
	}
	device, ok := mtl.CreateSystemDefaultDevice()
	if !ok {
		t.Skip("no Metal device")
	}
	defer mtl.Release(device)
	objc.AutoreleasePool(func() {
		options := mtl.NewCompileOptions()
		defer mtl.Release(options)
		library, err := device.NewLibraryWithSource(stitchedShaderSource(), options)
		if err != nil {
			t.Fatalf("building Metal library: %v", err)
		}
		defer mtl.Release(library)
		for _, pair := range shaderFunctionNames {
			for _, name := range pair {
				f, ok := library.NewFunctionWithName(name)
				if !ok {
					t.Errorf("locating function %s", name)
					continue
				}
				mtl.Release(f)
			}
		}
	})
}
