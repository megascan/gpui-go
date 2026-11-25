//go:build darwin

package mtl

import (
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

// Device wraps an id<MTLDevice>.
type Device objc.ID

// ID returns the underlying object.
func (d Device) ID() objc.ID { return objc.ID(d) }

// CreateSystemDefaultDevice returns the +1 default device, or false when the system has none.
func CreateSystemDefaultDevice() (Device, bool) {
	mustLoad()
	d := Device(mtlCreateSystemDefaultDevice())
	return d, d != 0
}

// CopyAllDevices returns every Metal device, each +1; it is empty when enumeration fails or the
// platform lacks MTLCopyAllDevices.
func CopyAllDevices() []Device {
	mustLoad()
	if mtlCopyAllDevices == nil {
		return nil
	}
	array := mtlCopyAllDevices()
	if array == objc.Nil {
		return nil
	}
	defer objc.Release(array)
	ids := objc.NSArrayToIDs(array)
	out := make([]Device, len(ids))
	for i, id := range ids {
		out[i] = Device(objc.Retain(id))
	}
	return out
}

// Name returns the device name.
func (d Device) Name() string { return objc.GoString(send[objc.ID](d.ID(), "name")) }

// IsRemovable reports whether the device is an external GPU.
func (d Device) IsRemovable() bool { return send[bool](d.ID(), "isRemovable") }

// IsLowPower reports whether the device is a low-power integrated GPU.
func (d Device) IsLowPower() bool { return send[bool](d.ID(), "isLowPower") }

// HasUnifiedMemory reports whether the CPU and GPU share memory.
func (d Device) HasUnifiedMemory() bool { return send[bool](d.ID(), "hasUnifiedMemory") }

// SupportsFamily reports whether the device supports the GPU family.
func (d Device) SupportsFamily(family GPUFamily) bool {
	return send[bool](d.ID(), "supportsFamily:", family)
}

// NewCommandQueue returns a +1 command queue, or 0 on failure.
func (d Device) NewCommandQueue() CommandQueue {
	return CommandQueue(send[objc.ID](d.ID(), "newCommandQueue"))
}

// NewBufferWithLength returns a +1 buffer of length bytes, or 0 on failure.
func (d Device) NewBufferWithLength(length uint, options ResourceOptions) Buffer {
	return Buffer(send[objc.ID](d.ID(), "newBufferWithLength:options:", length, options))
}

// NewTextureWithDescriptor returns a +1 texture, or 0 on failure.
func (d Device) NewTextureWithDescriptor(descriptor TextureDescriptor) Texture {
	return Texture(send[objc.ID](d.ID(), "newTextureWithDescriptor:", descriptor.ID()))
}

// NewSamplerStateWithDescriptor returns a +1 sampler state, or 0 on failure.
func (d Device) NewSamplerStateWithDescriptor(descriptor SamplerDescriptor) SamplerState {
	return SamplerState(send[objc.ID](d.ID(), "newSamplerStateWithDescriptor:", descriptor.ID()))
}

// NewDepthStencilStateWithDescriptor returns a +1 depth stencil state, or 0 on failure.
func (d Device) NewDepthStencilStateWithDescriptor(descriptor DepthStencilDescriptor) DepthStencilState {
	return DepthStencilState(send[objc.ID](d.ID(), "newDepthStencilStateWithDescriptor:", descriptor.ID()))
}

// NewLibraryWithSource compiles Metal Shading Language source into a +1 library.
func (d Device) NewLibraryWithSource(source string, options CompileOptions) (Library, error) {
	mustLoad()
	var library objc.ID
	var err error
	objc.AutoreleasePool(func() {
		var errID objc.ID
		library = msgSendIDIDPtrRetID(d.ID(), objc.Sel("newLibraryWithSource:options:error:"),
			objc.NSStringFromGo(source), options.ID(), unsafe.Pointer(&errID))
		if library == objc.Nil {
			err = nsError(errID)
			if err == nil {
				err = errNilResult("newLibraryWithSource:options:error:")
			}
		}
	})
	return Library(library), err
}

// NewRenderPipelineStateWithDescriptor returns a +1 render pipeline state.
func (d Device) NewRenderPipelineStateWithDescriptor(descriptor RenderPipelineDescriptor) (RenderPipelineState, error) {
	mustLoad()
	var state objc.ID
	var err error
	objc.AutoreleasePool(func() {
		var errID objc.ID
		state = msgSendIDPtrRetID(d.ID(), objc.Sel("newRenderPipelineStateWithDescriptor:error:"),
			descriptor.ID(), unsafe.Pointer(&errID))
		if state == objc.Nil {
			err = nsError(errID)
			if err == nil {
				err = errNilResult("newRenderPipelineStateWithDescriptor:error:")
			}
		}
	})
	return RenderPipelineState(state), err
}

type errNilResult string

func (e errNilResult) Error() string { return "mtl: " + string(e) + " returned nil" }

// CompileOptions wraps an MTLCompileOptions.
type CompileOptions objc.ID

// ID returns the underlying object.
func (o CompileOptions) ID() objc.ID { return objc.ID(o) }

// NewCompileOptions returns +1 default compile options.
func NewCompileOptions() CompileOptions { return CompileOptions(newObject("MTLCompileOptions")) }

// Library wraps an id<MTLLibrary>.
type Library objc.ID

// ID returns the underlying object.
func (l Library) ID() objc.ID { return objc.ID(l) }

// NewFunctionWithName returns the +1 function named name, or false when the library has none.
func (l Library) NewFunctionWithName(name string) (Function, bool) {
	var f objc.ID
	objc.AutoreleasePool(func() {
		f = send[objc.ID](l.ID(), "newFunctionWithName:", objc.NSStringFromGo(name))
	})
	return Function(f), f != objc.Nil
}

// FunctionNames returns the names of the library's functions.
func (l Library) FunctionNames() []string {
	var names []string
	objc.AutoreleasePool(func() {
		for _, id := range objc.NSArrayToIDs(send[objc.ID](l.ID(), "functionNames")) {
			names = append(names, objc.GoString(id))
		}
	})
	return names
}

// Function wraps an id<MTLFunction>.
type Function objc.ID

// ID returns the underlying object.
func (f Function) ID() objc.ID { return objc.ID(f) }

// Name returns the function name.
func (f Function) Name() string { return objc.GoString(send[objc.ID](f.ID(), "name")) }

// CommandQueue wraps an id<MTLCommandQueue>.
type CommandQueue objc.ID

// ID returns the underlying object.
func (q CommandQueue) ID() objc.ID { return objc.ID(q) }

// SetLabel sets the debug label.
func (q CommandQueue) SetLabel(label string) { setLabel(q.ID(), label) }

// CommandBuffer returns an autoreleased command buffer.
func (q CommandQueue) CommandBuffer() CommandBuffer {
	return CommandBuffer(send[objc.ID](q.ID(), "commandBuffer"))
}

// Buffer wraps an id<MTLBuffer>.
type Buffer objc.ID

// ID returns the underlying object.
func (b Buffer) ID() objc.ID { return objc.ID(b) }

// Contents returns the CPU address of the buffer's storage.
func (b Buffer) Contents() unsafe.Pointer {
	return send[unsafe.Pointer](b.ID(), "contents")
}

// Length returns the buffer size in bytes.
func (b Buffer) Length() uint { return send[uint](b.ID(), "length") }

// DidModifyRange tells Metal the CPU changed the bytes in r of a managed buffer.
func (b Buffer) DidModifyRange(r objc.NSRange) {
	mustLoad()
	msgSendNSRange(b.ID(), objc.Sel("didModifyRange:"), r)
}

// SetLabel sets the debug label.
func (b Buffer) SetLabel(label string) { setLabel(b.ID(), label) }

// SamplerDescriptor wraps an MTLSamplerDescriptor.
type SamplerDescriptor objc.ID

// ID returns the underlying object.
func (s SamplerDescriptor) ID() objc.ID { return objc.ID(s) }

// NewSamplerDescriptor returns a +1 sampler descriptor.
func NewSamplerDescriptor() SamplerDescriptor {
	return SamplerDescriptor(newObject("MTLSamplerDescriptor"))
}

// SetMinFilter sets the minification filter.
func (s SamplerDescriptor) SetMinFilter(f SamplerMinMagFilter) {
	sendVoid(s.ID(), "setMinFilter:", f)
}

// SetMagFilter sets the magnification filter.
func (s SamplerDescriptor) SetMagFilter(f SamplerMinMagFilter) {
	sendVoid(s.ID(), "setMagFilter:", f)
}

// SetSAddressMode sets the horizontal address mode.
func (s SamplerDescriptor) SetSAddressMode(m SamplerAddressMode) {
	sendVoid(s.ID(), "setSAddressMode:", m)
}

// SetTAddressMode sets the vertical address mode.
func (s SamplerDescriptor) SetTAddressMode(m SamplerAddressMode) {
	sendVoid(s.ID(), "setTAddressMode:", m)
}

// SamplerState wraps an id<MTLSamplerState>.
type SamplerState objc.ID

// ID returns the underlying object.
func (s SamplerState) ID() objc.ID { return objc.ID(s) }

// DepthStencilDescriptor wraps an MTLDepthStencilDescriptor.
type DepthStencilDescriptor objc.ID

// ID returns the underlying object.
func (d DepthStencilDescriptor) ID() objc.ID { return objc.ID(d) }

// NewDepthStencilDescriptor returns a +1 depth stencil descriptor.
func NewDepthStencilDescriptor() DepthStencilDescriptor {
	return DepthStencilDescriptor(newObject("MTLDepthStencilDescriptor"))
}

// SetDepthCompareFunction sets the depth test.
func (d DepthStencilDescriptor) SetDepthCompareFunction(f CompareFunction) {
	sendVoid(d.ID(), "setDepthCompareFunction:", f)
}

// SetDepthWriteEnabled sets whether passing fragments write depth.
func (d DepthStencilDescriptor) SetDepthWriteEnabled(enabled bool) {
	sendVoid(d.ID(), "setDepthWriteEnabled:", enabled)
}

// DepthStencilState wraps an id<MTLDepthStencilState>.
type DepthStencilState objc.ID

// ID returns the underlying object.
func (d DepthStencilState) ID() objc.ID { return objc.ID(d) }

// RenderPipelineState wraps an id<MTLRenderPipelineState>.
type RenderPipelineState objc.ID

// ID returns the underlying object.
func (s RenderPipelineState) ID() objc.ID { return objc.ID(s) }

// RenderPipelineDescriptor wraps an MTLRenderPipelineDescriptor.
type RenderPipelineDescriptor objc.ID

// ID returns the underlying object.
func (d RenderPipelineDescriptor) ID() objc.ID { return objc.ID(d) }

// NewRenderPipelineDescriptor returns a +1 render pipeline descriptor.
func NewRenderPipelineDescriptor() RenderPipelineDescriptor {
	return RenderPipelineDescriptor(newObject("MTLRenderPipelineDescriptor"))
}

// SetLabel sets the debug label.
func (d RenderPipelineDescriptor) SetLabel(label string) { setLabel(d.ID(), label) }

// SetVertexFunction sets the vertex shader.
func (d RenderPipelineDescriptor) SetVertexFunction(f Function) {
	sendVoid(d.ID(), "setVertexFunction:", f.ID())
}

// SetFragmentFunction sets the fragment shader.
func (d RenderPipelineDescriptor) SetFragmentFunction(f Function) {
	sendVoid(d.ID(), "setFragmentFunction:", f.ID())
}

// SetDepthAttachmentPixelFormat sets the depth attachment format.
func (d RenderPipelineDescriptor) SetDepthAttachmentPixelFormat(format PixelFormat) {
	sendVoid(d.ID(), "setDepthAttachmentPixelFormat:", format)
}

// SetRasterSampleCount sets the multisample count.
func (d RenderPipelineDescriptor) SetRasterSampleCount(count uint) {
	sendVoid(d.ID(), "setRasterSampleCount:", count)
}

// ColorAttachment returns the color attachment at index.
func (d RenderPipelineDescriptor) ColorAttachment(index uint) RenderPipelineColorAttachmentDescriptor {
	array := send[objc.ID](d.ID(), "colorAttachments")
	return RenderPipelineColorAttachmentDescriptor(send[objc.ID](array, "objectAtIndexedSubscript:", index))
}

// RenderPipelineColorAttachmentDescriptor wraps an MTLRenderPipelineColorAttachmentDescriptor.
type RenderPipelineColorAttachmentDescriptor objc.ID

// ID returns the underlying object.
func (a RenderPipelineColorAttachmentDescriptor) ID() objc.ID { return objc.ID(a) }

// SetPixelFormat sets the attachment format.
func (a RenderPipelineColorAttachmentDescriptor) SetPixelFormat(format PixelFormat) {
	sendVoid(a.ID(), "setPixelFormat:", format)
}

// SetBlendingEnabled sets whether blending is on.
func (a RenderPipelineColorAttachmentDescriptor) SetBlendingEnabled(enabled bool) {
	sendVoid(a.ID(), "setBlendingEnabled:", enabled)
}

// SetRgbBlendOperation sets the color blend operation.
func (a RenderPipelineColorAttachmentDescriptor) SetRgbBlendOperation(op BlendOperation) {
	sendVoid(a.ID(), "setRgbBlendOperation:", op)
}

// SetAlphaBlendOperation sets the alpha blend operation.
func (a RenderPipelineColorAttachmentDescriptor) SetAlphaBlendOperation(op BlendOperation) {
	sendVoid(a.ID(), "setAlphaBlendOperation:", op)
}

// SetSourceRGBBlendFactor sets the source color factor.
func (a RenderPipelineColorAttachmentDescriptor) SetSourceRGBBlendFactor(f BlendFactor) {
	sendVoid(a.ID(), "setSourceRGBBlendFactor:", f)
}

// SetSourceAlphaBlendFactor sets the source alpha factor.
func (a RenderPipelineColorAttachmentDescriptor) SetSourceAlphaBlendFactor(f BlendFactor) {
	sendVoid(a.ID(), "setSourceAlphaBlendFactor:", f)
}

// SetDestinationRGBBlendFactor sets the destination color factor.
func (a RenderPipelineColorAttachmentDescriptor) SetDestinationRGBBlendFactor(f BlendFactor) {
	sendVoid(a.ID(), "setDestinationRGBBlendFactor:", f)
}

// SetDestinationAlphaBlendFactor sets the destination alpha factor.
func (a RenderPipelineColorAttachmentDescriptor) SetDestinationAlphaBlendFactor(f BlendFactor) {
	sendVoid(a.ID(), "setDestinationAlphaBlendFactor:", f)
}
