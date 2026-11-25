//go:build js && wasm

package webgl2

import (
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

const maxBindGroups = 8

type command interface{ isCommand() }

type cmdBeginRenderPass struct {
	color      *textureView
	loadOp     gpu.LoadOp
	storeOp    gpu.StoreOp
	clearColor gpu.Color
	depth      *textureView
	depthLoad  gpu.LoadOp
	depthStore gpu.StoreOp
	depthClear float32
}

type cmdEndRenderPass struct {
	discardColor bool
	discardDepth bool
}

type cmdSetPipeline struct{ pipeline *renderPipeline }

type cmdSetBindGroup struct {
	index uint32
	group *bindGroup
}

type cmdSetViewport struct{ x, y, width, height, minDepth, maxDepth float32 }

type cmdSetScissor struct{ x, y, width, height uint32 }

type cmdDraw struct{ vertexCount, instanceCount, firstVertex, firstInstance uint32 }

type cmdCopyTextureToBuffer struct {
	source      gpu.TexelCopyTextureInfo
	destination gpu.TexelCopyBufferInfo
	size        gpu.Extent3d
}

func (cmdBeginRenderPass) isCommand()     {}
func (cmdEndRenderPass) isCommand()       {}
func (cmdSetPipeline) isCommand()         {}
func (cmdSetBindGroup) isCommand()        {}
func (cmdSetViewport) isCommand()         {}
func (cmdSetScissor) isCommand()          {}
func (cmdDraw) isCommand()                {}
func (cmdCopyTextureToBuffer) isCommand() {}

type commandEncoder struct {
	device   *device
	commands []command
}

func (e *commandEncoder) BeginRenderPass(descriptor *gpu.RenderPassDescriptor) gpu.RenderPass {
	begin := cmdBeginRenderPass{}
	end := cmdEndRenderPass{}
	if len(descriptor.ColorAttachments) > 1 {
		e.device.report(gpu.ErrorFilterValidation, "The WebGL2 backend supports one color attachment per render pass")
	}
	if len(descriptor.ColorAttachments) > 0 {
		a := descriptor.ColorAttachments[0]
		begin.color, _ = a.View.(*textureView)
		begin.loadOp, begin.storeOp, begin.clearColor = a.LoadOp, a.StoreOp, a.ClearValue
		end.discardColor = a.StoreOp == gpu.StoreOpDiscard
	}
	if ds := descriptor.DepthStencilAttachment; ds != nil {
		begin.depth, _ = ds.View.(*textureView)
		begin.depthLoad, begin.depthStore, begin.depthClear = ds.DepthLoadOp, ds.DepthStoreOp, ds.DepthClearValue
		end.discardDepth = ds.DepthStoreOp == gpu.StoreOpDiscard
	}
	e.commands = append(e.commands, begin)
	return &renderPass{encoder: e, end: end}
}

func (e *commandEncoder) CopyTextureToBuffer(source gpu.TexelCopyTextureInfo, destination gpu.TexelCopyBufferInfo, size gpu.Extent3d) {
	e.commands = append(e.commands, cmdCopyTextureToBuffer{source: source, destination: destination, size: size})
}

func (e *commandEncoder) Finish() gpu.CommandBuffer {
	cb := &commandBuffer{commands: e.commands}
	e.commands = nil
	return cb
}

func (e *commandEncoder) Release() {}

type commandBuffer struct {
	commands []command
}

func (c *commandBuffer) Release() {}

type renderPass struct {
	encoder *commandEncoder
	end     cmdEndRenderPass
	ended   bool
}

func (p *renderPass) push(c command) {
	if p.ended {
		panic("gpui: render pass used after End")
	}
	p.encoder.commands = append(p.encoder.commands, c)
}

func (p *renderPass) SetPipeline(pipeline gpu.RenderPipeline) {
	rp, _ := pipeline.(*renderPipeline)
	p.push(cmdSetPipeline{pipeline: rp})
}

func (p *renderPass) SetBindGroup(index uint32, group gpu.BindGroup, offsets []uint32) {
	bg, _ := group.(*bindGroup)
	if len(offsets) != 0 {
		p.encoder.device.report(gpu.ErrorFilterValidation, "Dynamic offsets are not used by gpui's bind groups")
	}
	p.push(cmdSetBindGroup{index: index, group: bg})
}

func (p *renderPass) SetViewport(x, y, width, height, minDepth, maxDepth float32) {
	p.push(cmdSetViewport{x, y, width, height, minDepth, maxDepth})
}

func (p *renderPass) SetScissorRect(x, y, width, height uint32) {
	p.push(cmdSetScissor{x, y, width, height})
}

func (p *renderPass) Draw(vertexCount, instanceCount, firstVertex, firstInstance uint32) {
	p.push(cmdDraw{vertexCount, instanceCount, firstVertex, firstInstance})
}

func (p *renderPass) End() {
	if p.ended {
		return
	}
	p.push(p.end)
	p.ended = true
}

type execState struct {
	valid          bool
	pipeline       *renderPipeline
	program        glObject
	primitive      primitiveState
	colorTargets   []colorTargetDesc
	blendEnabled   bool
	groups         [maxBindGroups]*bindGroup
	dirtyGroups    uint32
	appliedLayout  *pipelineLayout
	samplers       [maxSamplers]*sampler
	textureBound   [maxTextureSlots]bool
	dirtyTextures  uint32
	dirtySamplers  uint32
	passDepthBound bool
}

func (d *device) resetState() {
	gl := d.gl
	gl.useProgram(0)
	gl.bindFramebuffer(glFramebuffer, 0)
	gl.disable(glDepthTest)
	gl.disable(glStencilTest)
	gl.disable(glScissorTest)
	gl.disable(glBlend)
	gl.disable(glCullFace)
	d.state = execState{valid: true, primitive: primitiveState{frontFace: glCCW}}
}

func (d *device) execute(commands []command) {
	if len(commands) == 0 {
		return
	}
	d.resetState()
	for _, c := range commands {
		switch c := c.(type) {
		case cmdBeginRenderPass:
			d.beginRenderPass(c)
		case cmdEndRenderPass:
			d.endRenderPass(c)
		case cmdSetPipeline:
			d.setPipeline(c.pipeline)
		case cmdSetBindGroup:
			if c.index < maxBindGroups {
				d.state.groups[c.index] = c.group
				d.state.dirtyGroups |= 1 << c.index
			}
		case cmdSetViewport:
			d.gl.viewport(int(c.x), int(c.y), int(c.width), int(c.height))
			d.gl.depthRange(c.minDepth, c.maxDepth)
		case cmdSetScissor:
			d.gl.scissor(int(c.x), int(c.y), int(c.width), int(c.height))
		case cmdDraw:
			d.draw(c)
		case cmdCopyTextureToBuffer:
			d.copyTextureToBuffer(c)
		}
	}
}

func (d *device) beginRenderPass(c cmdBeginRenderPass) {
	gl := d.gl
	if d.drawFramebuffer == 0 {
		d.drawFramebuffer = gl.createFramebuffer()
	}
	gl.bindFramebuffer(glDrawFramebuffer, d.drawFramebuffer)
	var width, height uint32
	if c.color != nil && c.color.texture != nil {
		t := c.color.texture
		gl.framebufferTexture2D(glDrawFramebuffer, glColorAttachment0, glTexture2D, t.raw, 0)
		width, height = t.width, t.height
	} else {
		gl.framebufferTexture2D(glDrawFramebuffer, glColorAttachment0, glTexture2D, 0, 0)
	}
	if c.depth != nil && c.depth.texture != nil {
		gl.framebufferTexture2D(glDrawFramebuffer, glDepthAttachment, glTexture2D, c.depth.texture.raw, 0)
		d.state.passDepthBound = true
		if width == 0 {
			width, height = c.depth.texture.width, c.depth.texture.height
		}
	} else {
		gl.framebufferTexture2D(glDrawFramebuffer, glDepthAttachment, glTexture2D, 0, 0)
		d.state.passDepthBound = false
	}
	gl.drawBuffers([]uint32{glColorAttachment0})
	gl.disable(glScissorTest)
	if c.color != nil && c.loadOp == gpu.LoadOpClear {
		gl.colorMask(true, true, true, true)
		d.state.colorTargets = nil
		gl.clearBufferfv(glColor, 0, [4]float32{float32(c.clearColor.R), float32(c.clearColor.G), float32(c.clearColor.B), float32(c.clearColor.A)})
	}
	if c.depth != nil && c.depthLoad == gpu.LoadOpClear {
		gl.depthMask(true)
		gl.clearBufferfv(glDepth, 0, [4]float32{c.depthClear})
	}
	gl.viewport(0, 0, int(width), int(height))
	gl.depthRange(0, 1)
	gl.scissor(0, 0, int(width), int(height))
	gl.enable(glScissorTest)
}

func (d *device) endRenderPass(c cmdEndRenderPass) {
	gl := d.gl
	var attachments []uint32
	if c.discardColor {
		attachments = append(attachments, glColorAttachment0)
	}
	if c.discardDepth && d.state.passDepthBound {
		attachments = append(attachments, glDepthAttachment)
	}
	if len(attachments) > 0 {
		gl.invalidateFramebuffer(glDrawFramebuffer, attachments)
	}
	gl.bindFramebuffer(glDrawFramebuffer, 0)
}

func (d *device) setPipeline(p *renderPipeline) {
	if p == nil {
		d.state.pipeline = nil
		return
	}
	gl := d.gl
	s := &d.state
	s.pipeline = p
	if s.program != p.program {
		gl.useProgram(p.program)
		s.program = p.program
		s.dirtyTextures = ^uint32(0)
	}
	for i := range maxBindGroups {
		if s.appliedLayout == nil || p.layout == nil || !p.layout.sameGroup(i, s.appliedLayout) {
			s.dirtyGroups |= 1 << i
		}
	}

	if p.primitive != s.primitive {
		gl.frontFace(p.primitive.frontFace)
		if p.primitive.cullFace != 0 {
			gl.enable(glCullFace)
			gl.cullFace(p.primitive.cullFace)
		} else {
			gl.disable(glCullFace)
		}
		s.primitive = p.primitive
	}

	if p.depth != nil {
		gl.enable(glDepthTest)
		gl.depthFunc(p.depth.function)
		gl.depthMask(p.depth.mask)
	} else {
		gl.disable(glDepthTest)
	}

	if !colorTargetsEqual(s.colorTargets, p.colorTargets) {
		if len(p.colorTargets) > 0 {
			ct := p.colorTargets[0]
			gl.colorMask(ct.mask&gpu.ColorWritesRed != 0, ct.mask&gpu.ColorWritesGreen != 0, ct.mask&gpu.ColorWritesBlue != 0, ct.mask&gpu.ColorWritesAlpha != 0)
			if ct.blend != nil {
				gl.enable(glBlend)
				gl.blendEquationSeparate(ct.blend.opColor, ct.blend.opAlpha)
				gl.blendFuncSeparate(ct.blend.srcColor, ct.blend.dstColor, ct.blend.srcAlpha, ct.blend.dstAlpha)
			} else {
				gl.disable(glBlend)
			}
		}
		s.colorTargets = p.colorTargets
	}
}

func colorTargetsEqual(a, b []colorTargetDesc) bool {
	if a == nil || len(a) != len(b) {
		return false
	}
	for i := range a {
		if !a[i].equal(b[i]) {
			return false
		}
	}
	return true
}

func (d *device) applyBindGroups() {
	s := &d.state
	p := s.pipeline
	gl := d.gl
	if p.layout == nil {
		return
	}
	for index, info := range p.layout.groups {
		if index >= maxBindGroups || s.dirtyGroups&(1<<index) == 0 || info == nil {
			continue
		}
		group := s.groups[index]
		if group == nil {
			continue
		}
		s.dirtyGroups &^= 1 << index
		for i, entry := range info.layout.entries {
			if i >= len(group.contents) {
				break
			}
			res := group.contents[i]
			slot := info.bindingToSlot[entry.Binding]
			switch registerOf(entry) {
			case registerUniformBuffers:
				b, _ := res.Buffer.(*buffer)
				if b == nil {
					continue
				}
				size := res.Size
				if size == 0 {
					size = b.size - res.Offset
				}
				gl.bindBufferRange(glUniformBuffer, uint32(slot), b.raw, int(res.Offset), int(size))
			case registerTextures:
				v, _ := res.TextureView.(*textureView)
				var raw glObject
				if v != nil && v.texture != nil {
					raw = v.texture.raw
				}
				gl.activeTexture(glTexture0 + uint32(slot))
				gl.bindTexture(glTexture2D, raw)
				s.textureBound[slot] = true
				s.dirtyTextures |= 1 << slot
			case registerSamplers:
				smp, _ := res.Sampler.(*sampler)
				s.samplers[slot] = smp
				s.dirtySamplers |= 1 << slot
			}
		}
	}
	s.appliedLayout = p.layout
	d.rebindSamplerStates()
}

func (d *device) rebindSamplerStates() {
	s := &d.state
	p := s.pipeline
	for textureIndex := range maxTextureSlots {
		if !s.textureBound[textureIndex] {
			continue
		}
		samplerIndex := p.samplerMap[textureIndex]
		if s.dirtyTextures&(1<<textureIndex) == 0 && (samplerIndex < 0 || s.dirtySamplers&(1<<samplerIndex) == 0) {
			continue
		}
		var raw glObject
		if samplerIndex >= 0 {
			if smp := s.samplers[samplerIndex]; smp != nil {
				raw = smp.raw
			}
		}
		d.gl.bindSampler(uint32(textureIndex), raw)
	}
	s.dirtyTextures = 0
	s.dirtySamplers = 0
}

func (d *device) draw(c cmdDraw) {
	p := d.state.pipeline
	if p == nil || p.program == 0 {
		return
	}
	d.applyBindGroups()
	if p.firstInstanceLoc != 0 {
		d.gl.uniform1ui(p.firstInstanceLoc, c.firstInstance)
	}
	d.gl.drawArraysInstanced(p.topology, int(c.firstVertex), int(c.vertexCount), int(c.instanceCount))
}

func (d *device) copyTextureToBuffer(c cmdCopyTextureToBuffer) {
	t, _ := c.source.Texture.(*texture)
	b, _ := c.destination.Buffer.(*buffer)
	if t == nil || b == nil || t.raw == 0 || b.destroyed {
		d.report(gpu.ErrorFilterValidation, "Copy source or destination is invalid")
		return
	}
	bpp := t.formatDesc.bytesPerTexel
	rowBytes := c.size.Width * bpp
	stride := c.destination.Layout.BytesPerRow
	if stride == 0 {
		stride = rowBytes
	}
	if c.size.Width == 0 || c.size.Height == 0 {
		return
	}
	needed := uint64(stride)*uint64(c.size.Height-1) + uint64(rowBytes)
	offset := c.destination.Layout.Offset
	if offset+needed > b.size {
		d.report(gpu.ErrorFilterValidation, "Copy would overrun the destination buffer")
		return
	}
	gl := d.gl
	if d.readFramebuffer == 0 {
		d.readFramebuffer = gl.createFramebuffer()
	}
	gl.bindFramebuffer(glReadFramebuffer, d.readFramebuffer)
	gl.framebufferTexture2D(glReadFramebuffer, glColorAttachment0, glTexture2D, t.raw, int(c.source.MipLevel))
	gl.pixelStorei(glPackAlignment, 1)
	if stride != rowBytes {
		gl.pixelStorei(glPackRowLength, int(stride/bpp))
	}
	dst := make([]byte, needed)
	gl.readPixels(int(c.source.Origin.X), int(c.source.Origin.Y), int(c.size.Width), int(c.size.Height), t.formatDesc.external, t.formatDesc.dataType, dst)
	if stride != rowBytes {
		gl.pixelStorei(glPackRowLength, 0)
	}
	gl.bindFramebuffer(glReadFramebuffer, 0)
	if t.formatDesc.swizzleBgra {
		dst = swizzleRB(dst)
	}
	if b.shadow != nil {
		copy(b.shadow[offset:], dst)
	}
	if b.raw != 0 {
		gl.bindBuffer(glCopyWriteBuffer, b.raw)
		gl.bufferSubData(glCopyWriteBuffer, int(offset), dst)
		gl.bindBuffer(glCopyWriteBuffer, 0)
	}
}
