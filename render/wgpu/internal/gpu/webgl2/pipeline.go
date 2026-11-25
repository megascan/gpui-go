//go:build js && wasm

package webgl2

import (
	"errors"
	"fmt"

	"github.com/megascan/gpui-go/internal/webglsl"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

type bindingRegister uint8

const (
	registerUniformBuffers bindingRegister = iota
	registerStorageBuffers
	registerTextures
	registerSamplers
)

type groupInfo struct {
	layout        *bindGroupLayout
	bindingToSlot map[uint32]uint8
}

type pipelineLayout struct {
	groups []*groupInfo
}

func (l *pipelineLayout) Release() {}

func registerOf(entry gpu.BindGroupLayoutEntry) bindingRegister {
	switch {
	case entry.Type.Sampler != nil:
		return registerSamplers
	case entry.Type.Texture != nil:
		return registerTextures
	case entry.Type.Buffer != nil && entry.Type.Buffer.Type == gpu.BufferBindingTypeUniform:
		return registerUniformBuffers
	}
	return registerStorageBuffers
}

func createPipelineLayout(layouts []gpu.BindGroupLayout) (*pipelineLayout, error) {
	var counters [4]uint8
	out := &pipelineLayout{}
	var err error
	for _, l := range layouts {
		bgl, ok := l.(*bindGroupLayout)
		if !ok || bgl == nil {
			out.groups = append(out.groups, nil)
			continue
		}
		info := &groupInfo{layout: bgl, bindingToSlot: map[uint32]uint8{}}
		for _, entry := range bgl.entries {
			register := registerOf(entry)
			if register == registerStorageBuffers && err == nil {
				err = errors.New("Storage buffers are not supported by the WebGL2 backend")
			}
			info.bindingToSlot[entry.Binding] = counters[register]
			counters[register]++
		}
		out.groups = append(out.groups, info)
	}
	if err == nil && counters[registerTextures] > maxTextureSlots {
		err = fmt.Errorf("Too many bindings of type SampledTextures in Stage ShaderStages(VERTEX | FRAGMENT), limit is %d, count was %d", maxTextureSlots, counters[registerTextures])
	}
	if err == nil && counters[registerSamplers] > maxSamplers {
		err = fmt.Errorf("Too many bindings of type Samplers in Stage ShaderStages(VERTEX | FRAGMENT), limit is %d, count was %d", maxSamplers, counters[registerSamplers])
	}
	return out, err
}

func (l *pipelineLayout) slot(group, binding int) (uint8, bool) {
	if group < 0 || group >= len(l.groups) || l.groups[group] == nil {
		return 0, false
	}
	s, ok := l.groups[group].bindingToSlot[uint32(binding)]
	return s, ok
}

func (l *pipelineLayout) sameGroup(index int, other *pipelineLayout) bool {
	if other == nil || index >= len(l.groups) || index >= len(other.groups) {
		return false
	}
	a, b := l.groups[index], other.groups[index]
	if a == nil || b == nil {
		return a == b
	}
	if a.layout != b.layout || len(a.bindingToSlot) != len(b.bindingToSlot) {
		return false
	}
	for k, v := range a.bindingToSlot {
		if b.bindingToSlot[k] != v {
			return false
		}
	}
	return true
}

type renderPipeline struct {
	gl               glContext
	program          glObject
	layout           *pipelineLayout
	samplerMap       [maxTextureSlots]int8
	firstInstanceLoc glObject
	topology         uint32
	primitive        primitiveState
	depth            *depthState
	colorTargets     []colorTargetDesc
	released         bool
}

func (p *renderPipeline) Release() {
	if p.released {
		return
	}
	p.released = true
	if p.program != 0 {
		p.gl.deleteProgram(p.program)
	}
}

func findProgram(vertexEntry, fragmentEntry string) (webglsl.Program, bool) {
	for _, p := range webglsl.WebglPrograms() {
		if p.VertexEntry == vertexEntry && p.FragmentEntry == fragmentEntry {
			return p, true
		}
	}
	return webglsl.Program{}, false
}

func (d *device) compileShader(stage uint32, source, file string) (glObject, error) {
	gl := d.gl
	shader := gl.createShader(stage)
	gl.shaderSource(shader, source)
	gl.compileShader(shader)
	if !gl.getShaderCompileStatus(shader) {
		msg := gl.getShaderInfoLog(shader)
		gl.deleteShader(shader)
		return 0, fmt.Errorf("Shader compilation of %s failed: %s", file, msg)
	}
	return shader, nil
}

func (d *device) createRenderPipeline(descriptor *gpu.RenderPipelineDescriptor) (*renderPipeline, error) {
	p := &renderPipeline{
		gl:        d.gl,
		topology:  mapPrimitiveTopology(descriptor.Primitive.Topology),
		primitive: mapPrimitiveState(descriptor.Primitive),
		depth:     mapDepth(descriptor.DepthStencil),
	}
	for i := range p.samplerMap {
		p.samplerMap[i] = -1
	}
	layout, ok := descriptor.Layout.(*pipelineLayout)
	if !ok || layout == nil {
		return p, errors.New("Pipeline layout is invalid")
	}
	p.layout = layout
	if descriptor.Fragment != nil {
		for _, target := range descriptor.Fragment.Targets {
			p.colorTargets = append(p.colorTargets, mapColorTarget(target))
		}
	}
	vertexModule, _ := descriptor.Vertex.Module.(*shaderModule)
	if vertexModule == nil || !vertexModule.translated {
		return p, errors.New("Shader module is not the WebGL WGSL variant translated in internal/webglsl")
	}
	fragmentEntry := ""
	if descriptor.Fragment != nil {
		fragmentModule, _ := descriptor.Fragment.Module.(*shaderModule)
		if fragmentModule == nil || !fragmentModule.translated {
			return p, errors.New("Shader module is not the WebGL WGSL variant translated in internal/webglsl")
		}
		fragmentEntry = descriptor.Fragment.EntryPoint
	}
	program, ok := findProgram(descriptor.Vertex.EntryPoint, fragmentEntry)
	if !ok {
		return p, fmt.Errorf("Unable to find entry point '%s' / '%s' in the translated WebGL shaders", descriptor.Vertex.EntryPoint, fragmentEntry)
	}

	gl := d.gl
	vertex, err := d.compileShader(glVertexShader, program.Vertex.Source, program.Vertex.File)
	if err != nil {
		return p, err
	}
	fragment, err := d.compileShader(glFragmentShader, program.Fragment.Source, program.Fragment.File)
	if err != nil {
		gl.deleteShader(vertex)
		return p, err
	}
	raw := gl.createProgram()
	gl.attachShader(raw, vertex)
	gl.attachShader(raw, fragment)
	gl.linkProgram(raw)
	gl.deleteShader(vertex)
	gl.deleteShader(fragment)
	if !gl.getProgramLinkStatus(raw) {
		msg := gl.getProgramInfoLog(raw)
		gl.deleteProgram(raw)
		return p, fmt.Errorf("Pipeline linkage failed for stages ShaderStages(VERTEX | FRAGMENT): %s", msg)
	}

	gl.useProgram(raw)
	for _, stage := range []webglsl.Stage{program.Vertex, program.Fragment} {
		for _, block := range stage.UniformBlocks {
			slot, ok := layout.slot(block.Group, block.Binding)
			if !ok {
				gl.useProgram(0)
				gl.deleteProgram(raw)
				return p, fmt.Errorf("Shader global ResourceBinding { group: %d, binding: %d } is not available in the pipeline layout", block.Group, block.Binding)
			}
			if index := gl.getUniformBlockIndex(raw, block.Name); index != glInvalidIndex {
				gl.uniformBlockBinding(raw, index, uint32(slot))
			}
		}
		for _, tex := range stage.Textures {
			slot, ok := layout.slot(tex.Group, tex.Binding)
			if !ok {
				gl.useProgram(0)
				gl.deleteProgram(raw)
				return p, fmt.Errorf("Shader global ResourceBinding { group: %d, binding: %d } is not available in the pipeline layout", tex.Group, tex.Binding)
			}
			if location := gl.getUniformLocation(raw, tex.Name); location != 0 {
				gl.uniform1i(location, int32(slot))
			}
			if tex.SamplerGroup >= 0 {
				samplerSlot, ok := layout.slot(tex.SamplerGroup, tex.SamplerBinding)
				if !ok {
					gl.useProgram(0)
					gl.deleteProgram(raw)
					return p, fmt.Errorf("Shader global ResourceBinding { group: %d, binding: %d } is not available in the pipeline layout", tex.SamplerGroup, tex.SamplerBinding)
				}
				p.samplerMap[slot] = int8(samplerSlot)
			}
		}
	}
	p.firstInstanceLoc = gl.getUniformLocation(raw, "naga_vs_first_instance")
	gl.useProgram(0)
	p.program = raw
	d.state.valid = false
	return p, nil
}
