package webglsl

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"testing"
)

var rendererEntryPoints = [][3]string{
	{"quads", "vs_quad", "fs_quad"},
	{"opaque_quads", "vs_opaque_quad", "fs_opaque_quad"},
	{"shadows", "vs_shadow", "fs_shadow"},
	{"path_tiles", "vs_path_tile", "fs_path_tile"},
	{"underlines", "vs_underline", "fs_underline"},
	{"mono_sprites", "vs_mono_sprite", "fs_mono_sprite"},
	{"poly_sprites", "vs_poly_sprite", "fs_poly_sprite"},
	{"surfaces", "vs_surface", "fs_surface"},
}

func TestWebglsl_TableCoversRendererEntryPoints(t *testing.T) {
	if len(WebglPrograms()) != len(rendererEntryPoints) {
		t.Fatalf("got %d programs, want %d", len(WebglPrograms()), len(rendererEntryPoints))
	}
	for _, want := range rendererEntryPoints {
		p, ok := ProgramByName(want[0])
		if !ok {
			t.Fatalf("missing program %s", want[0])
		}
		if p.VertexEntry != want[1] || p.FragmentEntry != want[2] {
			t.Errorf("%s: entries %s/%s, want %s/%s", want[0], p.VertexEntry, p.FragmentEntry, want[1], want[2])
		}
		for _, st := range []Stage{p.Vertex, p.Fragment} {
			if !strings.HasPrefix(st.Source, "#version 300 es\n") {
				t.Errorf("%s: source does not start with #version 300 es", st.File)
			}
			for _, u := range st.UniformBlocks {
				if !strings.Contains(st.Source, "uniform "+u.Name+" ") {
					t.Errorf("%s: uniform block %s not in source", st.File, u.Name)
				}
			}
			for _, tex := range st.Textures {
				if tex.Unit != tex.Group*16+tex.Binding || !strings.Contains(st.Source, " "+tex.Name+";") {
					t.Errorf("%s: bad texture %+v", st.File, tex)
				}
			}
		}
	}
}

func TestWebglsl_SourceHashMatchesRendererWGSL(t *testing.T) {
	var src []byte
	for _, f := range []string{"shaders.wgsl", "shaders_webgl.wgsl"} {
		b, err := os.ReadFile("../../render/wgpu/shaders/" + f)
		if err != nil {
			t.Fatal(err)
		}
		src = append(src, b...)
	}
	sum := sha256.Sum256(src)
	if got := hex.EncodeToString(sum[:]); got != WebglWGSLSHA256 {
		t.Fatalf("WGSL changed (%s != %s); run go generate ./internal/webglsl", got, WebglWGSLSHA256)
	}
}
