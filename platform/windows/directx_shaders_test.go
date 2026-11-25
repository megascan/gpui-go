//go:build windows

package windows

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/internal/win/d3d"
)

var upstreamShaderHashes = map[string]string{
	"shaders.hlsl":          "fbfce2de66d9f28316e0ee91ea4d7af73055540d242804bab613ef2daa8d1588",
	"alpha_correction.hlsl": "d37b106f1266b40a0e17478455dfaac1205214f597e23fbd0a26455c49806bf4",
}

func TestDirectxShaders_MatchUpstream(t *testing.T) {
	embedded := map[string][]byte{
		"shaders.hlsl":          shadersHLSL,
		"alpha_correction.hlsl": alphaCorrectionHLSL,
	}
	refDir := filepath.Join("..", "..", "_ref", "gpui_windows", "src")
	_, refErr := os.Stat(refDir)
	for name, source := range embedded {
		sum := sha256.Sum256(source)
		if got := hex.EncodeToString(sum[:]); got != upstreamShaderHashes[name] {
			t.Errorf("%s sha256 %s, want %s", name, got, upstreamShaderHashes[name])
		}
		if refErr != nil {
			continue
		}
		ref, err := os.ReadFile(filepath.Join(refDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if sha256.Sum256(ref) != sum {
			t.Errorf("%s differs from upstream", name)
		}
	}
}

func TestDirectxShaders_CompileEveryEntryPoint(t *testing.T) {
	modules := []ShaderModule{
		ShaderModuleQuad,
		ShaderModuleOpaqueQuad,
		ShaderModuleShadow,
		ShaderModulePathTile,
		ShaderModuleUnderline,
		ShaderModuleMonochromeSprite,
		ShaderModuleSubpixelSprite,
		ShaderModulePolychromeSprite,
	}
	for _, module := range modules {
		for _, target := range []ShaderTarget{ShaderTargetVertex, ShaderTargetFragment} {
			raw, err := NewRawShaderBytes(module, target)
			if err != nil {
				t.Errorf("%s target %d: %v", module.AsStr(), target, err)
				continue
			}
			if len(raw.AsBytes()) == 0 {
				t.Errorf("%s target %d: empty bytecode", module.AsStr(), target)
			}
		}
	}
}

func TestDirectxShaders_IncludeServesOnlyAlphaCorrection(t *testing.T) {
	source := []byte("#include \"other.hlsl\"\nfloat4 main_ps() : SV_Target { return 0; }\n")
	code, errs, err := d3d.D3DCompile(source, "shaders.hlsl", shaderInclude(), "main_ps", "ps_4_1", 0, 0)
	if err == nil {
		code.Release()
		t.Fatal("an include other than alpha_correction.hlsl was served")
	}
	if errs != nil {
		msg := errs.String()
		errs.Release()
		if !strings.Contains(msg, "other.hlsl") {
			t.Fatalf("error %q does not name the include", msg)
		}
	}
}

func TestDirectxShaders_CreateShadersOnWarp(t *testing.T) {
	device, _, deviceContext, err := d3d.D3D11CreateDevice(nil, d3d.D3D_DRIVER_TYPE_WARP, 0, d3d.D3D11_CREATE_DEVICE_BGRA_SUPPORT, nil, d3d.D3D11_SDK_VERSION)
	if err != nil {
		t.Skipf("no WARP device: %v", err)
	}
	defer device.Release()
	defer deviceContext.Release()
	vs, err := NewRawShaderBytes(ShaderModuleQuad, ShaderTargetVertex)
	if err != nil {
		t.Fatal(err)
	}
	vertex, err := device.CreateVertexShader(vs.AsBytes())
	if err != nil {
		t.Fatal(err)
	}
	vertex.Release()
	ps, err := NewRawShaderBytes(ShaderModuleQuad, ShaderTargetFragment)
	if err != nil {
		t.Fatal(err)
	}
	pixel, err := device.CreatePixelShader(ps.AsBytes())
	if err != nil {
		t.Fatal(err)
	}
	pixel.Release()
}
