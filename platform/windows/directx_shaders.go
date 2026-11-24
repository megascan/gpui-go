//go:build windows

package windows

import (
	_ "embed"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/win/d3d"
)

//go:embed shaders.hlsl
var shadersHLSL []byte

//go:embed alpha_correction.hlsl
var alphaCorrectionHLSL []byte

// ShaderModule is one of the renderer's shader programs.
type ShaderModule uint8

// ShaderModule values.
const (
	ShaderModuleQuad ShaderModule = iota
	ShaderModuleOpaqueQuad
	ShaderModuleShadow
	ShaderModuleUnderline
	ShaderModulePathTile
	ShaderModuleMonochromeSprite
	ShaderModuleSubpixelSprite
	ShaderModulePolychromeSprite
)

// AsStr returns the module's entry point prefix in shaders.hlsl.
func (m ShaderModule) AsStr() string {
	switch m {
	case ShaderModuleQuad:
		return "quad"
	case ShaderModuleOpaqueQuad:
		return "opaque_quad"
	case ShaderModuleShadow:
		return "shadow"
	case ShaderModuleUnderline:
		return "underline"
	case ShaderModulePathTile:
		return "path_tile"
	case ShaderModuleMonochromeSprite:
		return "monochrome_sprite"
	case ShaderModuleSubpixelSprite:
		return "subpixel_sprite"
	case ShaderModulePolychromeSprite:
		return "polychrome_sprite"
	}
	panic("internal error: entered unreachable code")
}

// ShaderTarget is the pipeline stage a shader is compiled for.
type ShaderTarget uint8

// ShaderTarget values.
const (
	ShaderTargetVertex ShaderTarget = iota
	ShaderTargetFragment
)

// RawShaderBytes is compiled shader bytecode.
type RawShaderBytes struct {
	inner []byte
}

type shaderKey struct {
	module ShaderModule
	target ShaderTarget
}

var compiledShaders sync.Map

// NewRawShaderBytes returns the bytecode of module for target, compiling shaders.hlsl with
// D3DCompile on first use, where upstream's release build embeds the output of fxc.
func NewRawShaderBytes(module ShaderModule, target ShaderTarget) (*RawShaderBytes, error) {
	key := shaderKey{module: module, target: target}
	if bytes, ok := compiledShaders.Load(key); ok {
		return &RawShaderBytes{inner: bytes.([]byte)}, nil
	}
	bytes, err := buildShaderBlob(module, target)
	if err != nil {
		return nil, err
	}
	actual, _ := compiledShaders.LoadOrStore(key, bytes)
	return &RawShaderBytes{inner: actual.([]byte)}, nil
}

// AsBytes returns the bytecode.
func (r *RawShaderBytes) AsBytes() []byte {
	return r.inner
}

var shaderInclude = sync.OnceValue(func() *d3d.Include {
	return d3d.NewInclude(map[string][]byte{"alpha_correction.hlsl": alphaCorrectionHLSL})
})

func shaderCompileFlags() uint32 {
	if gpuiutil.DebugAssertions {
		return d3d.D3DCOMPILE_DEBUG | d3d.D3DCOMPILE_SKIP_OPTIMIZATION
	}
	return d3d.D3DCOMPILE_OPTIMIZATION_LEVEL3
}

func buildShaderBlob(entry ShaderModule, target ShaderTarget) ([]byte, error) {
	var suffix, profile string
	switch target {
	case ShaderTargetVertex:
		suffix, profile = "vertex", "vs_4_1"
	case ShaderTargetFragment:
		suffix, profile = "fragment", "ps_4_1"
	default:
		panic("internal error: entered unreachable code")
	}
	entryPoint := entry.AsStr() + "_" + suffix
	compileBlob, errorBlob, err := d3d.D3DCompile(shadersHLSL, "shaders.hlsl", shaderInclude(), entryPoint, profile, shaderCompileFlags(), 0)
	if err != nil {
		if errorBlob == nil {
			return nil, err
		}
		errorString := errorBlob.String()
		errorBlob.Release()
		slog.Error(fmt.Sprintf("Shader compile error: %s", errorString))
		return nil, errors.New("Compile error: " + errorString)
	}
	if errorBlob != nil {
		errorBlob.Release()
	}
	defer compileBlob.Release()
	return append([]byte(nil), compileBlob.Bytes()...), nil
}
