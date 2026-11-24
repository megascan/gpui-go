//go:build darwin

package apple

import _ "embed"

//go:generate go run github.com/megascan/gpui-go/internal/cmd/metalhdr -ref ../../_ref -out shaders_scene_gen.go

//go:embed shaders.metal
var shadersMetal string

var shaderFunctionNames = [...][2]string{
	{"path_tile_vertex", "path_tile_fragment"},
	{"shadow_vertex", "shadow_fragment"},
	{"quad_vertex", "quad_fragment"},
	{"opaque_quad_vertex", "opaque_quad_fragment"},
	{"underline_vertex", "underline_fragment"},
	{"monochrome_sprite_vertex", "monochrome_sprite_fragment"},
	{"polychrome_sprite_vertex", "polychrome_sprite_fragment"},
	{"surface_vertex", "surface_fragment"},
	{"native_view_vertex", "native_view_fragment"},
}

func stitchedShaderSource() string {
	return sceneHeader + "\n" + shadersMetal
}
