// Package webglsl holds the GLSL ES 3.00 shaders the WebGL2 renderer path uses, translated by naga from render/wgpu's WGSL.
package webglsl

//go:generate go run github.com/megascan/gpui-go/internal/cmd/glslgen -shaders ../../render/wgpu/shaders -out .

// UniformBlock is a std140 uniform block naga emitted for a WGSL uniform binding.
type UniformBlock struct {
	Name     string
	Instance string
	Group    int
	Binding  int
}

// Texture is a combined sampler uniform naga emitted for a WGSL texture binding; SamplerGroup and SamplerBinding are -1 when the texture is only fetched.
type Texture struct {
	Name           string
	Group          int
	Binding        int
	Unit           int
	SamplerGroup   int
	SamplerBinding int
}

// Stage is one translated shader stage with the names the GL backend binds by.
type Stage struct {
	File          string
	Source        string
	UniformBlocks []UniformBlock
	Textures      []Texture
	Uniforms      []string
	Attributes    []string
}

// Program pairs the vertex and fragment stages of one renderer pipeline.
type Program struct {
	Name          string
	VertexEntry   string
	FragmentEntry string
	Vertex        Stage
	Fragment      Stage
}

// WebglPrograms returns the programs of the WebGL shader variant (shaders.wgsl + shaders_webgl.wgsl).
func WebglPrograms() []Program {
	return webglPrograms
}

// ProgramByName returns the WebGL program for a pipeline name.
func ProgramByName(name string) (Program, bool) {
	for _, p := range webglPrograms {
		if p.Name == name {
			return p, true
		}
	}
	return Program{}, false
}
