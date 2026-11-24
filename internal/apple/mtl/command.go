//go:build darwin

package mtl

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// CommandBuffer wraps an id<MTLCommandBuffer>.
type CommandBuffer objc.ID

// ID returns the underlying object.
func (b CommandBuffer) ID() objc.ID { return objc.ID(b) }

// SetLabel sets the debug label.
func (b CommandBuffer) SetLabel(label string) { setLabel(b.ID(), label) }

// RenderCommandEncoderWithDescriptor returns an autoreleased render encoder for the pass.
func (b CommandBuffer) RenderCommandEncoderWithDescriptor(descriptor RenderPassDescriptor) RenderCommandEncoder {
	return RenderCommandEncoder(send[objc.ID](b.ID(), "renderCommandEncoderWithDescriptor:", descriptor.ID()))
}

// BlitCommandEncoder returns an autoreleased blit encoder.
func (b CommandBuffer) BlitCommandEncoder() BlitCommandEncoder {
	return BlitCommandEncoder(send[objc.ID](b.ID(), "blitCommandEncoder"))
}

// PresentDrawable schedules drawable to be presented when the buffer completes.
func (b CommandBuffer) PresentDrawable(drawable Drawable) {
	sendVoid(b.ID(), "presentDrawable:", drawable.ID())
}

// AddCompletedHandler registers handler to run when the GPU finishes the buffer.
func (b CommandBuffer) AddCompletedHandler(handler func(CommandBuffer)) {
	block := objc.NewBlock(func(_ objc.Block, buffer objc.ID) {
		handler(CommandBuffer(buffer))
	})
	sendVoid(b.ID(), "addCompletedHandler:", block)
	block.Release()
}

// Commit submits the buffer.
func (b CommandBuffer) Commit() { sendVoid(b.ID(), "commit") }

// WaitUntilScheduled blocks until the buffer is scheduled.
func (b CommandBuffer) WaitUntilScheduled() { sendVoid(b.ID(), "waitUntilScheduled") }

// WaitUntilCompleted blocks until the GPU finishes the buffer.
func (b CommandBuffer) WaitUntilCompleted() { sendVoid(b.ID(), "waitUntilCompleted") }

// Status returns the buffer's state.
func (b CommandBuffer) Status() CommandBufferStatus {
	return send[CommandBufferStatus](b.ID(), "status")
}

// Error returns the error that failed the buffer, or nil.
func (b CommandBuffer) Error() error { return nsError(send[objc.ID](b.ID(), "error")) }

// RenderCommandEncoder wraps an id<MTLRenderCommandEncoder>.
type RenderCommandEncoder objc.ID

// ID returns the underlying object.
func (e RenderCommandEncoder) ID() objc.ID { return objc.ID(e) }

// SetLabel sets the debug label.
func (e RenderCommandEncoder) SetLabel(label string) { setLabel(e.ID(), label) }

// EndEncoding finishes the encoder.
func (e RenderCommandEncoder) EndEncoding() { sendVoid(e.ID(), "endEncoding") }

// SetRenderPipelineState sets the pipeline for later draws.
func (e RenderCommandEncoder) SetRenderPipelineState(state RenderPipelineState) {
	sendVoid(e.ID(), "setRenderPipelineState:", state.ID())
}

// SetDepthStencilState sets the depth state for later draws.
func (e RenderCommandEncoder) SetDepthStencilState(state DepthStencilState) {
	sendVoid(e.ID(), "setDepthStencilState:", state.ID())
}

// SetViewport sets the viewport.
func (e RenderCommandEncoder) SetViewport(viewport Viewport) {
	mustLoad()
	msgSendViewport(e.ID(), objc.Sel("setViewport:"), viewport)
}

// SetVertexBuffer binds buffer at offset to vertex argument index.
func (e RenderCommandEncoder) SetVertexBuffer(buffer Buffer, offset, index uint) {
	mustLoad()
	msgSendIDUintUint(e.ID(), objc.Sel("setVertexBuffer:offset:atIndex:"), buffer.ID(), offset, index)
}

// SetVertexBytes copies length bytes at bytes to vertex argument index.
func (e RenderCommandEncoder) SetVertexBytes(bytes unsafe.Pointer, length, index uint) {
	mustLoad()
	msgSendPtrUintUint(e.ID(), objc.Sel("setVertexBytes:length:atIndex:"), bytes, length, index)
}

// SetVertexTexture binds texture to vertex texture index.
func (e RenderCommandEncoder) SetVertexTexture(texture Texture, index uint) {
	mustLoad()
	msgSendIDUint(e.ID(), objc.Sel("setVertexTexture:atIndex:"), texture.ID(), index)
}

// SetFragmentBuffer binds buffer at offset to fragment argument index.
func (e RenderCommandEncoder) SetFragmentBuffer(buffer Buffer, offset, index uint) {
	mustLoad()
	msgSendIDUintUint(e.ID(), objc.Sel("setFragmentBuffer:offset:atIndex:"), buffer.ID(), offset, index)
}

// SetFragmentBytes copies length bytes at bytes to fragment argument index.
func (e RenderCommandEncoder) SetFragmentBytes(bytes unsafe.Pointer, length, index uint) {
	mustLoad()
	msgSendPtrUintUint(e.ID(), objc.Sel("setFragmentBytes:length:atIndex:"), bytes, length, index)
}

// SetFragmentTexture binds texture to fragment texture index.
func (e RenderCommandEncoder) SetFragmentTexture(texture Texture, index uint) {
	mustLoad()
	msgSendIDUint(e.ID(), objc.Sel("setFragmentTexture:atIndex:"), texture.ID(), index)
}

// SetFragmentSamplerState binds sampler to fragment sampler index.
func (e RenderCommandEncoder) SetFragmentSamplerState(sampler SamplerState, index uint) {
	mustLoad()
	msgSendIDUint(e.ID(), objc.Sel("setFragmentSamplerState:atIndex:"), sampler.ID(), index)
}

// DrawPrimitivesInstanced draws instanceCount instances of vertexCount vertices.
func (e RenderCommandEncoder) DrawPrimitivesInstanced(primitive PrimitiveType, vertexStart, vertexCount, instanceCount uint) {
	mustLoad()
	msgSendUint4(e.ID(), objc.Sel("drawPrimitives:vertexStart:vertexCount:instanceCount:"),
		uint(primitive), vertexStart, vertexCount, instanceCount)
}

// DrawPrimitives draws instanceCount instances of vertexCount vertices starting at baseInstance.
func (e RenderCommandEncoder) DrawPrimitives(primitive PrimitiveType, vertexStart, vertexCount, instanceCount, baseInstance uint) {
	mustLoad()
	msgSendUint5(e.ID(), objc.Sel("drawPrimitives:vertexStart:vertexCount:instanceCount:baseInstance:"),
		uint(primitive), vertexStart, vertexCount, instanceCount, baseInstance)
}

// BlitCommandEncoder wraps an id<MTLBlitCommandEncoder>.
type BlitCommandEncoder objc.ID

// ID returns the underlying object.
func (e BlitCommandEncoder) ID() objc.ID { return objc.ID(e) }

// SynchronizeResource makes GPU writes to a managed resource visible to the CPU.
func (e BlitCommandEncoder) SynchronizeResource(resource objc.ID) {
	sendVoid(e.ID(), "synchronizeResource:", resource)
}

// EndEncoding finishes the encoder.
func (e BlitCommandEncoder) EndEncoding() { sendVoid(e.ID(), "endEncoding") }

// RenderPassDescriptor wraps an MTLRenderPassDescriptor.
type RenderPassDescriptor objc.ID

// ID returns the underlying object.
func (d RenderPassDescriptor) ID() objc.ID { return objc.ID(d) }

// NewRenderPassDescriptor returns a +1 render pass descriptor.
func NewRenderPassDescriptor() RenderPassDescriptor {
	return RenderPassDescriptor(newObject("MTLRenderPassDescriptor"))
}

// ColorAttachment returns the color attachment at index.
func (d RenderPassDescriptor) ColorAttachment(index uint) RenderPassAttachmentDescriptor {
	array := send[objc.ID](d.ID(), "colorAttachments")
	return RenderPassAttachmentDescriptor(send[objc.ID](array, "objectAtIndexedSubscript:", index))
}

// DepthAttachment returns the depth attachment.
func (d RenderPassDescriptor) DepthAttachment() RenderPassAttachmentDescriptor {
	return RenderPassAttachmentDescriptor(send[objc.ID](d.ID(), "depthAttachment"))
}

// RenderPassAttachmentDescriptor wraps an MTLRenderPassColorAttachmentDescriptor or
// MTLRenderPassDepthAttachmentDescriptor.
type RenderPassAttachmentDescriptor objc.ID

// ID returns the underlying object.
func (a RenderPassAttachmentDescriptor) ID() objc.ID { return objc.ID(a) }

// SetTexture sets the attachment texture.
func (a RenderPassAttachmentDescriptor) SetTexture(texture Texture) {
	sendVoid(a.ID(), "setTexture:", texture.ID())
}

// SetResolveTexture sets the multisample resolve texture.
func (a RenderPassAttachmentDescriptor) SetResolveTexture(texture Texture) {
	sendVoid(a.ID(), "setResolveTexture:", texture.ID())
}

// SetLoadAction sets what happens to the attachment at the start of the pass.
func (a RenderPassAttachmentDescriptor) SetLoadAction(action LoadAction) {
	sendVoid(a.ID(), "setLoadAction:", action)
}

// SetStoreAction sets what happens to the attachment at the end of the pass.
func (a RenderPassAttachmentDescriptor) SetStoreAction(action StoreAction) {
	sendVoid(a.ID(), "setStoreAction:", action)
}

// SetClearColor sets the clear color of a color attachment.
func (a RenderPassAttachmentDescriptor) SetClearColor(color ClearColor) {
	mustLoad()
	msgSendClearColor(a.ID(), objc.Sel("setClearColor:"), color)
}

// SetClearDepth sets the clear depth of a depth attachment.
func (a RenderPassAttachmentDescriptor) SetClearDepth(depth float64) {
	sendVoid(a.ID(), "setClearDepth:", depth)
}
