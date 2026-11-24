//go:build js && wasm

package webgl2

import (
	"strconv"
	"strings"
	"testing"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

type testRig struct {
	gl       *fakeGL
	instance *instance
	surface  *surface
	adapter  gpu.Adapter
	device   *device
	queue    gpu.Queue
}

func newTestRig(t *testing.T) *testRig {
	t.Helper()
	gl := newFakeGL()
	inst := CreateInstance(gpu.InstanceDescriptor{Backends: gpu.BackendsGl}).(*instance)
	s := inst.newSurface(gl)
	a, err := inst.RequestAdapter(&gpu.RequestAdapterOptions{CompatibleSurface: s})
	if err != nil {
		t.Fatal(err)
	}
	d, q, err := a.RequestDevice(&gpu.DeviceDescriptor{RequiredLimits: gpu.DownlevelWebGL2Defaults().UsingResolution(a.Limits()).UsingAlignment(a.Limits())})
	if err != nil {
		t.Fatal(err)
	}
	return &testRig{gl: gl, instance: inst, surface: s, adapter: a, device: d.(*device), queue: q}
}

type rendererLayouts struct {
	globals, instances, texture, surfaces gpu.BindGroupLayout
}

func (r *testRig) layouts() rendererLayouts {
	d := r.device
	uniform := func(b uint32) gpu.BindGroupLayoutEntry {
		return gpu.BindGroupLayoutEntry{Binding: b, Visibility: gpu.ShaderStagesVertexFragment, Type: gpu.BindingType{Buffer: &gpu.BufferBindingLayout{Type: gpu.BufferBindingTypeUniform}}}
	}
	tex := func(b uint32, st gpu.TextureSampleType) gpu.BindGroupLayoutEntry {
		return gpu.BindGroupLayoutEntry{Binding: b, Visibility: gpu.ShaderStagesVertexFragment, Type: gpu.BindingType{Texture: &gpu.TextureBindingLayout{SampleType: st, ViewDimension: gpu.TextureViewDimensionD2}}}
	}
	smp := func(b uint32) gpu.BindGroupLayoutEntry {
		return gpu.BindGroupLayoutEntry{Binding: b, Visibility: gpu.ShaderStagesVertexFragment, Type: gpu.BindingType{Sampler: &gpu.SamplerBindingLayout{Type: gpu.SamplerBindingTypeFiltering}}}
	}
	return rendererLayouts{
		globals:   d.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{Entries: []gpu.BindGroupLayoutEntry{uniform(0), uniform(1)}}),
		instances: d.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{Entries: []gpu.BindGroupLayoutEntry{tex(0, gpu.TextureSampleTypeUint)}}),
		texture:   d.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{Entries: []gpu.BindGroupLayoutEntry{tex(0, gpu.TextureSampleTypeFloat), smp(1)}}),
		surfaces:  d.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{Entries: []gpu.BindGroupLayoutEntry{uniform(0), tex(1, gpu.TextureSampleTypeFloat), tex(2, gpu.TextureSampleTypeFloat), smp(3)}}),
	}
}

func (r *testRig) pipeline(t *testing.T, vs, fs string, blend *gpu.BlendState, groups ...gpu.BindGroupLayout) *renderPipeline {
	t.Helper()
	module := &shaderModule{translated: true}
	var scopeErr *gpu.Error
	r.device.PushErrorScope(gpu.ErrorFilterValidation)
	p := r.device.CreateRenderPipeline(&gpu.RenderPipelineDescriptor{
		Layout:       r.device.CreatePipelineLayout(&gpu.PipelineLayoutDescriptor{BindGroupLayouts: groups}),
		Vertex:       gpu.VertexState{Module: module, EntryPoint: vs},
		Fragment:     &gpu.FragmentState{Module: module, EntryPoint: fs, Targets: []gpu.ColorTargetState{{Format: gpu.TextureFormatRgba8Unorm, Blend: blend, WriteMask: gpu.ColorWritesAll}}},
		Primitive:    gpu.PrimitiveState{Topology: gpu.PrimitiveTopologyTriangleStrip, FrontFace: gpu.FrontFaceCcw, CullMode: gpu.CullModeNone},
		DepthStencil: &gpu.DepthStencilState{Format: gpu.TextureFormatDepth32Float, DepthCompare: gpu.CompareFunctionGreater},
	})
	scopeErr = r.device.PopErrorScope()
	if scopeErr != nil {
		t.Fatalf("pipeline %s/%s: %s", vs, fs, scopeErr.Message)
	}
	return p.(*renderPipeline)
}

func TestWebgl2_TextureFormatMapping(t *testing.T) {
	cases := []struct {
		format                     gpu.TextureFormat
		internal, external, dtype  uint32
		bpp                        uint32
		swizzle, filterable, depth bool
	}{
		{gpu.TextureFormatR8Unorm, glR8, glRed, glUnsignedByte, 1, false, true, false},
		{gpu.TextureFormatRgba8Unorm, glRGBA8, glRGBA, glUnsignedByte, 4, false, true, false},
		{gpu.TextureFormatRgba8UnormSrgb, glSRGB8Alpha8, glRGBA, glUnsignedByte, 4, false, true, false},
		{gpu.TextureFormatBgra8Unorm, glRGBA8, glRGBA, glUnsignedByte, 4, true, true, false},
		{gpu.TextureFormatBgra8UnormSrgb, glSRGB8Alpha8, glRGBA, glUnsignedByte, 4, true, true, false},
		{gpu.TextureFormatRgba32Uint, glRGBA32UI, glRGBAInteger, glUnsignedInt, 16, false, false, false},
		{gpu.TextureFormatDepth32Float, glDepthComponent32F, glDepthComponent, glFloat, 4, false, false, true},
	}
	for _, c := range cases {
		d, ok := describeTextureFormat(c.format)
		if !ok {
			t.Fatalf("%s: not described", c.format)
		}
		if d.internal != c.internal || d.external != c.external || d.dataType != c.dtype || d.bytesPerTexel != c.bpp || d.swizzleBgra != c.swizzle || d.filterable != c.filterable || d.depth != c.depth {
			t.Errorf("%s: got %+v", c.format, d)
		}
	}
	if _, ok := describeTextureFormat(gpu.TextureFormat(0x99)); ok {
		t.Error("unknown format described")
	}
}

func TestWebgl2_StateMapping(t *testing.T) {
	if min, mag := mapFilterModes(gpu.FilterModeLinear, gpu.FilterModeLinear, 0); min != glLinearMipmapNearest || mag != glLinear {
		t.Errorf("linear filter modes = %#x %#x", min, mag)
	}
	if mapAddressMode(0) != glClampToEdge || mapAddressMode(gpu.AddressModeRepeat) != glRepeat {
		t.Error("address modes")
	}
	if mapCompareFunc(gpu.CompareFunctionGreater) != glGreater || mapPrimitiveTopology(gpu.PrimitiveTopologyTriangleStrip) != glTriangleStrip {
		t.Error("compare / topology")
	}
	if ps := mapPrimitiveState(gpu.PrimitiveState{FrontFace: gpu.FrontFaceCcw}); ps.frontFace != glCW || ps.cullFace != 0 {
		t.Errorf("ccw front face must map to GL CW under the flipped coordinate space, got %+v", ps)
	}
	ct := mapColorTarget(gpu.ColorTargetState{Blend: &gpu.BlendStatePremultipliedAlphaBlending, WriteMask: gpu.ColorWritesAll})
	if *ct.blend != (blendDesc{glOne, glOneMinusSrcAlpha, glOne, glOneMinusSrcAlpha, glFuncAdd, glFuncAdd}) {
		t.Errorf("premultiplied blend = %+v", *ct.blend)
	}
}

func TestWebgl2_AdapterSelectsTextureInstanceData(t *testing.T) {
	r := newTestRig(t)
	info := r.adapter.GetInfo()
	if info.Backend != gpu.BackendGl || info.Name != "WebKit WebGL" {
		t.Errorf("info = %+v", info)
	}
	caps := r.adapter.GetDownlevelCapabilities()
	if caps.Flags.Contains(gpu.DownlevelFlagsVertexStorage) || caps.Flags.Contains(gpu.DownlevelFlagsFragmentStorage) {
		t.Error("WebGL2 adapter must not report storage buffers")
	}
	if r.adapter.Features().Contains(gpu.FeaturesDualSourceBlending) {
		t.Error("WebGL2 adapter must not report dual-source blending")
	}
	l := r.adapter.Limits()
	if l.MaxTextureDimension2D != 4096 || l.MinUniformBufferOffsetAlignment != 256 || l.MaxStorageBuffersPerShaderStage != 0 {
		t.Errorf("limits = %+v", l)
	}
	want := gpu.TextureUsagesTextureBinding | gpu.TextureUsagesCopyDst
	if !r.adapter.GetTextureFormatFeatures(gpu.TextureFormatRgba8Unorm).AllowedUsages.Contains(want) {
		t.Error("Rgba8Unorm must allow texture binding and copy dst")
	}
	caps2 := r.surface.GetCapabilities(r.adapter)
	if len(caps2.Formats) != 2 || caps2.Formats[0] != gpu.TextureFormatRgba8Unorm || caps2.AlphaModes[0] != gpu.CompositeAlphaModeOpaque {
		t.Errorf("surface caps = %+v", caps2)
	}
	if _, _, err := r.adapter.RequestDevice(&gpu.DeviceDescriptor{RequiredFeatures: gpu.FeaturesDualSourceBlending}); err == nil {
		t.Error("requesting dual-source blending must fail")
	}
	unmasked := newFakeGL()
	unmasked.extensions = []string{"WEBGL_debug_renderer_info"}
	unmasked.strings[glUnmaskedVendorWebGL] = "NVIDIA Corporation"
	unmasked.strings[glUnmaskedRendererWebGL] = "ANGLE (NVIDIA GeForce RTX)"
	if info := exposeAdapter(unmasked).GetInfo(); !unmasked.has("getExtension WEBGL_debug_renderer_info") || info.Name != "ANGLE (NVIDIA GeForce RTX)" || info.Vendor != 0x10DE {
		t.Errorf("unmasked adapter info = %+v", info)
	}
	if !r.gl.has("bindVertexArray " + itoa(int(r.device.mainVAO))) {
		t.Error("device must bind its main vertex array")
	}
	if got := makeInfo("Intel Inc.", "Intel Iris", "").DeviceType; got != gpu.DeviceTypeIntegratedGpu {
		t.Errorf("intel device type = %s", got)
	}
	if got := makeInfo("Google Inc.", "ANGLE (SwiftShader)", ""); got.DeviceType != gpu.DeviceTypeCpu || got.Vendor != 0 {
		t.Errorf("swiftshader info = %+v", got)
	}
	if got := makeInfo("NVIDIA Corporation", "GeForce", "").Vendor; got != 0x10DE {
		t.Errorf("nvidia vendor = %#x", got)
	}
}

func TestWebgl2_PipelineLayoutSlots(t *testing.T) {
	r := newTestRig(t)
	ls := r.layouts()
	sprites, err := createPipelineLayout([]gpu.BindGroupLayout{ls.globals, ls.instances, ls.texture})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ group, binding, slot int }{{0, 0, 0}, {0, 1, 1}, {1, 0, 0}, {2, 0, 1}, {2, 1, 0}} {
		if s, ok := sprites.slot(c.group, c.binding); !ok || int(s) != c.slot {
			t.Errorf("sprites (%d,%d) = %d %t, want %d", c.group, c.binding, s, ok, c.slot)
		}
	}
	surfaces, err := createPipelineLayout([]gpu.BindGroupLayout{ls.globals, ls.surfaces})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ group, binding, slot int }{{1, 0, 2}, {1, 1, 0}, {1, 2, 1}, {1, 3, 0}} {
		if s, ok := surfaces.slot(c.group, c.binding); !ok || int(s) != c.slot {
			t.Errorf("surfaces (%d,%d) = %d %t, want %d", c.group, c.binding, s, ok, c.slot)
		}
	}
	if !sprites.sameGroup(0, surfaces) || sprites.sameGroup(1, surfaces) {
		t.Error("sameGroup")
	}
	storage := r.device.CreateBindGroupLayout(&gpu.BindGroupLayoutDescriptor{Entries: []gpu.BindGroupLayoutEntry{{Type: gpu.BindingType{Buffer: &gpu.BufferBindingLayout{Type: gpu.BufferBindingTypeReadOnlyStorage}}}}})
	if _, err := createPipelineLayout([]gpu.BindGroupLayout{storage}); err == nil {
		t.Error("storage buffer layout must be rejected")
	}
}

func TestWebgl2_ProgramBindingTable(t *testing.T) {
	r := newTestRig(t)
	ls := r.layouts()
	p := r.pipeline(t, "vs_mono_sprite", "fs_mono_sprite", &gpu.BlendStateAlphaBlending, ls.globals, ls.instances, ls.texture)
	for _, want := range []string{
		"uniform1i _group_2_binding_0_vs 1",
		"uniform1i _group_2_binding_0_fs 1",
		"uniform1i _group_1_binding_0_vs 0",
		"getUniformBlockIndex GlobalParams_block_0Vertex",
		"getUniformBlockIndex GlobalParams_block_0Fragment",
		"getUniformBlockIndex GammaParams_block_1Fragment",
	} {
		if !r.gl.has(want) {
			t.Errorf("missing %q in %v", want, r.gl.calls)
		}
	}
	gamma := uint32(len("GammaParams_block_1Fragment") % 4)
	if !r.gl.has(strings.Join([]string{"uniformBlockBinding", itoa(int(p.program)), itoa(int(gamma)), "1"}, " ")) {
		t.Errorf("GammaParams must bind to uniform slot 1: %v", r.gl.calls)
	}
	if p.samplerMap[1] != 0 || p.samplerMap[0] != -1 {
		t.Errorf("sampler map = %v", p.samplerMap[:2])
	}
	if p.firstInstanceLoc == 0 || r.gl.locationName(p.firstInstanceLoc) != "naga_vs_first_instance" {
		t.Error("first instance uniform not looked up")
	}
	for _, prog := range [][2]string{{"vs_quad", "fs_quad"}, {"vs_opaque_quad", "fs_opaque_quad"}, {"vs_shadow", "fs_shadow"}, {"vs_path_tile", "fs_path_tile"}, {"vs_underline", "fs_underline"}, {"vs_poly_sprite", "fs_poly_sprite"}} {
		groups := []gpu.BindGroupLayout{ls.globals, ls.instances}
		if prog[0] == "vs_poly_sprite" {
			groups = append(groups, ls.texture)
		}
		r.pipeline(t, prog[0], prog[1], &gpu.BlendStateAlphaBlending, groups...)
	}
	r.pipeline(t, "vs_surface", "fs_surface", &gpu.BlendStateAlphaBlending, ls.globals, ls.surfaces)
}

func itoa(v int) string { return strconv.Itoa(v) }

func TestWebgl2_PipelineErrors(t *testing.T) {
	r := newTestRig(t)
	ls := r.layouts()
	layout := r.device.CreatePipelineLayout(&gpu.PipelineLayoutDescriptor{BindGroupLayouts: []gpu.BindGroupLayout{ls.globals, ls.instances}})
	r.device.PushErrorScope(gpu.ErrorFilterValidation)
	r.device.CreateRenderPipeline(&gpu.RenderPipelineDescriptor{Layout: layout, Vertex: gpu.VertexState{Module: r.device.CreateShaderModule(&gpu.ShaderModuleDescriptor{WGSL: "fn main() {}"}), EntryPoint: "vs_quad"}})
	if err := r.device.PopErrorScope(); err == nil || !strings.Contains(err.Message, "not the WebGL WGSL variant") {
		t.Errorf("untranslated module err = %v", err)
	}
	r.gl.failCompile = true
	r.device.PushErrorScope(gpu.ErrorFilterValidation)
	module := &shaderModule{translated: true}
	r.device.CreateRenderPipeline(&gpu.RenderPipelineDescriptor{Layout: layout, Vertex: gpu.VertexState{Module: module, EntryPoint: "vs_quad"}, Fragment: &gpu.FragmentState{Module: module, EntryPoint: "fs_quad"}})
	if err := r.device.PopErrorScope(); err == nil || !strings.Contains(err.Message, "syntax error") {
		t.Errorf("compile failure err = %v", err)
	}
	var uncaptured *gpu.Error
	r.device.OnUncapturedError(func(err *gpu.Error) { uncaptured = err })
	r.device.CreateRenderPipeline(&gpu.RenderPipelineDescriptor{Layout: layout, Vertex: gpu.VertexState{Module: module, EntryPoint: "vs_missing"}, Fragment: &gpu.FragmentState{Module: module, EntryPoint: "fs_quad"}})
	if uncaptured == nil || !strings.Contains(uncaptured.Message, "vs_missing") {
		t.Errorf("uncaptured = %v", uncaptured)
	}
	if err := r.device.PopErrorScope(); err == nil {
		t.Error("popping an empty scope stack must fail")
	}
}

type frame struct {
	globals   gpu.BindGroup
	instances gpu.BindGroup
	atlas     gpu.BindGroup
	sampler   *sampler
	color     gpu.TextureView
	depth     gpu.TextureView
}

func (r *testRig) frame(ls rendererLayouts) frame {
	d := r.device
	globals := d.CreateBuffer(&gpu.BufferDescriptor{Size: 512, Usage: gpu.BufferUsagesUniform | gpu.BufferUsagesCopyDst})
	instanceTex := d.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 64, Height: 64, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRgba32Uint, Dimension: gpu.TextureDimensionD2, MipLevelCount: 1})
	atlasTex := d.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 32, Height: 32, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatR8Unorm, Dimension: gpu.TextureDimensionD2, MipLevelCount: 1})
	smp := d.CreateSampler(&gpu.SamplerDescriptor{MagFilter: gpu.FilterModeLinear, MinFilter: gpu.FilterModeLinear})
	color := d.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 100, Height: 50, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRgba8Unorm, Dimension: gpu.TextureDimensionD2, MipLevelCount: 1})
	depth := d.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 100, Height: 50, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatDepth32Float, Dimension: gpu.TextureDimensionD2, MipLevelCount: 1})
	return frame{
		globals: d.CreateBindGroup(&gpu.BindGroupDescriptor{Layout: ls.globals, Entries: []gpu.BindGroupEntry{
			{Binding: 0, Resource: gpu.BindingResource{Buffer: globals, Size: 32}},
			{Binding: 1, Resource: gpu.BindingResource{Buffer: globals, Offset: 256, Size: 32}},
		}}),
		instances: d.CreateBindGroup(&gpu.BindGroupDescriptor{Layout: ls.instances, Entries: []gpu.BindGroupEntry{{Binding: 0, Resource: gpu.BindingResource{TextureView: instanceTex.CreateView(nil)}}}}),
		atlas: d.CreateBindGroup(&gpu.BindGroupDescriptor{Layout: ls.texture, Entries: []gpu.BindGroupEntry{
			{Binding: 0, Resource: gpu.BindingResource{TextureView: atlasTex.CreateView(nil)}},
			{Binding: 1, Resource: gpu.BindingResource{Sampler: smp}},
		}}),
		sampler: smp.(*sampler),
		color:   color.CreateView(nil),
		depth:   depth.CreateView(nil),
	}
}

func TestWebgl2_PipelineStateDiffingAndDraws(t *testing.T) {
	r := newTestRig(t)
	ls := r.layouts()
	quads := r.pipeline(t, "vs_quad", "fs_quad", &gpu.BlendStateAlphaBlending, ls.globals, ls.instances)
	shadows := r.pipeline(t, "vs_shadow", "fs_shadow", &gpu.BlendStateAlphaBlending, ls.globals, ls.instances)
	mono := r.pipeline(t, "vs_mono_sprite", "fs_mono_sprite", &gpu.BlendStatePremultipliedAlphaBlending, ls.globals, ls.instances, ls.texture)
	f := r.frame(ls)

	enc := r.device.CreateCommandEncoder(nil)
	pass := enc.BeginRenderPass(&gpu.RenderPassDescriptor{
		ColorAttachments:       []gpu.RenderPassColorAttachment{{View: f.color, LoadOp: gpu.LoadOpClear, StoreOp: gpu.StoreOpStore, ClearValue: gpu.Color{R: 1, A: 1}}},
		DepthStencilAttachment: &gpu.RenderPassDepthStencilAttachment{View: f.depth, DepthLoadOp: gpu.LoadOpClear, DepthStoreOp: gpu.StoreOpDiscard},
	})
	pass.SetBindGroup(0, f.globals, nil)
	pass.SetPipeline(quads)
	pass.SetBindGroup(1, f.instances, nil)
	pass.Draw(4, 3, 0, 0)
	pass.SetPipeline(shadows)
	pass.SetBindGroup(1, f.instances, nil)
	pass.Draw(4, 2, 0, 7)
	pass.SetPipeline(mono)
	pass.SetBindGroup(1, f.instances, nil)
	pass.SetBindGroup(2, f.atlas, nil)
	pass.SetViewport(0, 0, 100, 50, 0.25, 0.25)
	pass.Draw(4, 1, 0, 9)
	pass.End()
	cb := enc.Finish()
	r.gl.reset()
	r.queue.Submit(cb)

	if n := r.gl.count("blendFuncSeparate"); n != 2 {
		t.Errorf("blendFuncSeparate issued %d times, want 2 (quads, then mono's premultiplied blend): %v", n, r.gl.calls)
	}
	if n := r.gl.count("frontFace"); n != 1 {
		t.Errorf("frontFace issued %d times, want 1", n)
	}
	if n := r.gl.count("useProgram"); n != 4 {
		t.Errorf("useProgram issued %d times, want 4 (reset + three programs)", n)
	}
	for _, want := range []string{
		"clearBufferfv 0x1800 0 [1 0 0 1]",
		"clearBufferfv 0x1801 0 [0 0 0 0]",
		"bindBufferRange 0x8a11 0 " + itoa(int(f.globals.(*bindGroup).contents[0].Buffer.(*buffer).raw)) + " 0 32",
		"bindBufferRange 0x8a11 1 " + itoa(int(f.globals.(*bindGroup).contents[0].Buffer.(*buffer).raw)) + " 256 32",
		"uniform1ui naga_vs_first_instance 0",
		"uniform1ui naga_vs_first_instance 7",
		"uniform1ui naga_vs_first_instance 9",
		"drawArraysInstanced 0x5 0 4 3",
		"drawArraysInstanced 0x5 0 4 2",
		"bindSampler 0 0",
		"bindSampler 1 " + itoa(int(f.sampler.raw)),
		"depthRange 0.25 0.25",
		"invalidateFramebuffer [0x8d00]",
		"viewport 0 0 100 50",
	} {
		if !r.gl.has(want) {
			t.Errorf("missing %q", want)
		}
	}
	if n := r.gl.count("bindBufferRange"); n != 2 {
		t.Errorf("globals rebound %d times, want 2: the bind group layout did not change between pipelines", n)
	}
}

func TestWebgl2_SurfaceConfigurePresentFlip(t *testing.T) {
	r := newTestRig(t)
	r.surface.Configure(r.device, &gpu.SurfaceConfiguration{Format: gpu.TextureFormatRgba8Unorm, Width: 800, Height: 600, AlphaMode: gpu.CompositeAlphaModeOpaque, PresentMode: gpu.PresentModeFifo})
	if !r.gl.has("canvas 800x600") || !r.gl.has("texStorage2D 0xde1 1 0x8058 800 600") {
		t.Fatalf("configure calls: %v", r.gl.calls)
	}
	cur := r.surface.GetCurrentTexture()
	if cur.Status != gpu.CurrentSurfaceTextureSuccess || cur.Texture.Width() != 800 || cur.Texture.Format() != gpu.TextureFormatRgba8Unorm {
		t.Fatalf("current texture = %+v", cur)
	}
	cur.Texture.Destroy()
	r.gl.reset()
	r.surface.Present()
	if !r.gl.has("blitFramebuffer 0 600 800 0 0 0 800 600") {
		t.Errorf("present must blit with a Y flip: %v", r.gl.calls)
	}
	if r.gl.count("deleteTexture") != 0 {
		t.Error("destroying the surface texture must not delete the swapchain texture")
	}
	r.surface.Configure(r.device, &gpu.SurfaceConfiguration{Format: gpu.TextureFormatRgba8UnormSrgb, Width: 10, Height: 20})
	r.gl.reset()
	r.surface.Present()
	if !r.gl.has("drawArrays 0x4 0 3") || r.gl.has("blitFramebuffer 0 20 10 0 0 0 10 20") {
		t.Errorf("sRGB present must draw the srgb_present program: %v", r.gl.calls)
	}
	empty := r.instance.newSurface(r.gl)
	if empty.GetCurrentTexture().Status != gpu.CurrentSurfaceTextureValidation {
		t.Error("unconfigured surface must not return a texture")
	}
}

func TestWebgl2_WriteTextureSwizzleAndRowLength(t *testing.T) {
	r := newTestRig(t)
	tex := r.device.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 4, Height: 4, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatBgra8Unorm, MipLevelCount: 1})
	r.gl.reset()
	data := []byte{1, 2, 3, 4, 9, 9, 9, 9, 5, 6, 7, 8}
	r.queue.WriteTexture(gpu.TexelCopyTextureInfo{Texture: tex, Origin: gpu.Origin3d{X: 1, Y: 2}}, data, gpu.TexelCopyBufferLayout{BytesPerRow: 8}, gpu.Extent3d{Width: 1, Height: 2, DepthOrArrayLayers: 1})
	for _, want := range []string{
		"pixelStorei 0xcf5 1",
		"pixelStorei 0xcf2 2",
		"texSubImage2D 1 2 1 2 0x1908 0x1401 [3 2 1 4 9 9 9 9 7 6 5 8]",
		"pixelStorei 0xcf2 0",
	} {
		if !r.gl.has(want) {
			t.Errorf("missing %q in %v", want, r.gl.calls)
		}
	}
	inst := r.device.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 8, Height: 1, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRgba32Uint, MipLevelCount: 1})
	_ = inst
	if !r.gl.has("texParameteri 0xde1 0x2801 0x2600") || !r.gl.has("texParameteri 0xde1 0x2800 0x2600") {
		t.Error("integer textures must use NEAREST filtering")
	}
	var reported *gpu.Error
	r.device.OnUncapturedError(func(e *gpu.Error) { reported = e })
	r.queue.WriteTexture(gpu.TexelCopyTextureInfo{Texture: tex, Origin: gpu.Origin3d{X: 3}}, data, gpu.TexelCopyBufferLayout{}, gpu.Extent3d{Width: 2, Height: 1, DepthOrArrayLayers: 1})
	if reported == nil {
		t.Error("out-of-bounds write must report a validation error")
	}
}

func TestWebgl2_ReadbackThroughMapAsync(t *testing.T) {
	r := newTestRig(t)
	tex := r.device.CreateTexture(&gpu.TextureDescriptor{Size: gpu.Extent3d{Width: 2, Height: 2, DepthOrArrayLayers: 1}, Format: gpu.TextureFormatRgba8Unorm, MipLevelCount: 1})
	buf := r.device.CreateBuffer(&gpu.BufferDescriptor{Size: 512, Usage: gpu.BufferUsagesCopyDst | gpu.BufferUsagesMapRead})
	enc := r.device.CreateCommandEncoder(nil)
	enc.CopyTextureToBuffer(gpu.TexelCopyTextureInfo{Texture: tex}, gpu.TexelCopyBufferInfo{Buffer: buf, Layout: gpu.TexelCopyBufferLayout{BytesPerRow: 256, RowsPerImage: 2}}, gpu.Extent3d{Width: 2, Height: 2, DepthOrArrayLayers: 1})
	r.queue.Submit(enc.Finish())
	if !r.gl.has("pixelStorei 0xd02 64") || !r.gl.has("readPixels 0 0 2 2 264") {
		t.Fatalf("readback calls: %v", r.gl.calls)
	}
	var mapped bool
	buf.MapAsync(gpu.MapModeRead, 0, 512, func(err error) { mapped = err == nil })
	if mapped {
		t.Fatal("map callback must wait for Poll")
	}
	if err := r.device.Poll(true); err != nil || !mapped {
		t.Fatalf("poll err=%v mapped=%t", err, mapped)
	}
	got := buf.GetMappedRange(0, 512)
	if got[0] != 0 || got[7] != 7 || got[256] != 0 {
		t.Errorf("mapped data = %v %v %v", got[0], got[7], got[256])
	}
	buf.Unmap()
	if buf.GetMappedRange(0, 4) != nil {
		t.Error("unmapped buffer must not return a range")
	}
}

func TestWebgl2_DeviceLostListenerReleased(t *testing.T) {
	r := newTestRig(t)
	var reason gpu.DeviceLostReason = 99
	r.device.SetDeviceLostCallback(func(rs gpu.DeviceLostReason, msg string) { reason = rs })
	r.gl.lostCallback()
	if reason != gpu.DeviceLostReasonUnknown {
		t.Errorf("reason = %s", reason)
	}
	r.device.Release()
	r.device.Release()
	if r.gl.released != 1 {
		t.Errorf("context-lost listener released %d times, want 1", r.gl.released)
	}
}
