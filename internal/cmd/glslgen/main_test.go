package main

import "testing"

func TestGlslgen_Reflect(t *testing.T) {
	glsl := "#version 300 es\nuniform uint naga_vs_first_instance;\nlayout(std140) uniform GlobalParams_block_0Fragment { GlobalParams _group_0_binding_0_fs; };\nuniform highp sampler2D _group_2_binding_0_fs;\nuniform highp usampler2D _group_1_binding_0_fs;\nlayout(location = 0) in vec2 _p2vs_location0;\n"
	st := reflect(glsl, true, map[binding]binding{{2, 0}: {2, 1}})
	if len(st.UniformBlocks) != 1 || st.UniformBlocks[0].Name != "GlobalParams_block_0Fragment" || st.UniformBlocks[0].Instance != "_group_0_binding_0_fs" {
		t.Fatalf("blocks %+v", st.UniformBlocks)
	}
	if len(st.Textures) != 2 || st.Textures[0].SamplerBinding != 1 || st.Textures[1].SamplerGroup != -1 {
		t.Fatalf("textures %+v", st.Textures)
	}
	if len(st.Uniforms) != 1 || st.Uniforms[0] != "naga_vs_first_instance" {
		t.Fatalf("uniforms %v", st.Uniforms)
	}
	if len(st.Attributes) != 1 || st.Attributes[0] != "_p2vs_location0" {
		t.Fatalf("attributes %v", st.Attributes)
	}
}

func TestGlslgen_SamplerPairs(t *testing.T) {
	wgsl := "@group(2) @binding(0) var t_sprite: texture_2d<f32>;\n@group(2) @binding(1) var s_sprite: sampler;\nfn f() { let c = textureSample(t_sprite, s_sprite, uv); }\n"
	p := samplerPairs(wgsl)
	if p[binding{2, 0}] != (binding{2, 1}) {
		t.Fatalf("pairs %v", p)
	}
}
