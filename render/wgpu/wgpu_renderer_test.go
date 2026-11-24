//go:build linux

package wgpu

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func validateWGSL(t *testing.T, source string, features gpu.Features) {
	t.Helper()
	device, _ := testDeviceAndQueue(t, features)
	device.PushErrorScope(gpu.ErrorFilterValidation)
	module := device.CreateShaderModule(&gpu.ShaderModuleDescriptor{Label: "validate", WGSL: source})
	if err := device.PopErrorScope(); err != nil {
		t.Fatalf("shader should validate: %v", err)
	}
	module.Release()
}

func TestWgpuRenderer_WebglShaderIsValidWgslWithoutStorageBuffers(t *testing.T) {
	if strings.Contains(webglShaders, "var<storage") {
		t.Fatal("WebGL shaders must not use storage buffers")
	}
	validateWGSL(t, webglShaders, 0)
}

func TestWgpuRenderer_StorageBufferShaderIsValidWgsl(t *testing.T) {
	validateWGSL(t, storageBufferShaders, 0)
}

func TestWgpuRenderer_SubpixelStorageBufferShaderIsValidWgsl(t *testing.T) {
	validateWGSL(t, subpixelStorageBufferShaders, gpu.FeaturesDualSourceBlending)
}

func TestWgpuRenderer_SubpixelWebglShaderIsValidWgslWithoutStorageBuffers(t *testing.T) {
	if strings.Contains(subpixelWebglShaders, "var<storage") {
		t.Fatal("WebGL shaders must not use storage buffers")
	}
	validateWGSL(t, subpixelWebglShaders, gpu.FeaturesDualSourceBlending)
}

func TestShaders_MatchUpstream(t *testing.T) {
	refDir := filepath.Join("..", "..", "_ref", "gpui_wgpu", "src")
	if _, err := os.Stat(refDir); err != nil {
		t.Skipf("upstream reference absent: %v", err)
	}
	embedded := map[string]string{
		"shaders.wgsl":          shadersWGSL,
		"shaders_storage.wgsl":  shadersStorageWGSL,
		"shaders_webgl.wgsl":    shadersWebglWGSL,
		"shaders_subpixel.wgsl": shadersSubpixelWGSL,
	}
	for name, source := range embedded {
		ref, err := os.ReadFile(filepath.Join(refDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if sha256.Sum256(ref) != sha256.Sum256([]byte(source)) {
			t.Errorf("%s differs from upstream", name)
		}
	}
}
