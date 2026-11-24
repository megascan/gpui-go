//go:build js && wasm

// Package webgl2 implements the gpu facade over a WebGL2RenderingContext, mapped the way
// wgpu-hal's GLES backend maps the WebGPU API onto WebGL2.
package webgl2

type glObject uint32

const (
	glNoError                  = 0
	glDepthBufferBit           = 0x00000100
	glColorBufferBit           = 0x00004000
	glPoints                   = 0x0000
	glLines                    = 0x0001
	glLineStrip                = 0x0003
	glTriangles                = 0x0004
	glTriangleStrip            = 0x0005
	glZero                     = 0
	glOne                      = 1
	glSrcColor                 = 0x0300
	glOneMinusSrcColor         = 0x0301
	glSrcAlpha                 = 0x0302
	glOneMinusSrcAlpha         = 0x0303
	glDstAlpha                 = 0x0304
	glOneMinusDstAlpha         = 0x0305
	glDstColor                 = 0x0306
	glOneMinusDstColor         = 0x0307
	glFuncAdd                  = 0x8006
	glMin                      = 0x8007
	glMax                      = 0x8008
	glFuncSubtract             = 0x800A
	glFuncReverseSubtract      = 0x800B
	glFront                    = 0x0404
	glBack                     = 0x0405
	glCullFace                 = 0x0B44
	glDepthTest                = 0x0B71
	glStencilTest              = 0x0B90
	glBlend                    = 0x0BE2
	glScissorTest              = 0x0C11
	glUnpackAlignment          = 0x0CF5
	glPackAlignment            = 0x0D05
	glMaxTextureSize           = 0x0D33
	glTexture2D                = 0x0DE1
	glUnsignedByte             = 0x1401
	glUnsignedInt              = 0x1405
	glFloat                    = 0x1406
	glDepthComponent           = 0x1902
	glRed                      = 0x1903
	glRGBA                     = 0x1908
	glVendor                   = 0x1F00
	glRenderer                 = 0x1F01
	glVersion                  = 0x1F02
	glNearest                  = 0x2600
	glLinear                   = 0x2601
	glNearestMipmapNearest     = 0x2700
	glLinearMipmapNearest      = 0x2701
	glNearestMipmapLinear      = 0x2702
	glLinearMipmapLinear       = 0x2703
	glTextureMagFilter         = 0x2800
	glTextureMinFilter         = 0x2801
	glTextureWrapS             = 0x2802
	glTextureWrapT             = 0x2803
	glRepeat                   = 0x2901
	glCW                       = 0x0900
	glCCW                      = 0x0901
	glNever                    = 0x0200
	glLess                     = 0x0201
	glEqual                    = 0x0202
	glLequal                   = 0x0203
	glGreater                  = 0x0204
	glNotequal                 = 0x0205
	glGequal                   = 0x0206
	glAlways                   = 0x0207
	glTexture0                 = 0x84C0
	glClampToEdge              = 0x812F
	glMirroredRepeat           = 0x8370
	glTextureWrapR             = 0x8072
	glTextureMinLod            = 0x813A
	glTextureMaxLod            = 0x813B
	glTextureMaxLevel          = 0x813D
	glTextureCompareMode       = 0x884C
	glTextureCompareFunc       = 0x884D
	glCompareRefToTexture      = 0x884E
	glUnpackRowLength          = 0x0CF2
	glPackRowLength            = 0x0D02
	glR8                       = 0x8229
	glRGBA8                    = 0x8058
	glSRGB8Alpha8              = 0x8C43
	glRGBA32UI                 = 0x8D70
	glRGBAInteger              = 0x8D99
	glDepthComponent32F        = 0x8CAC
	glReadFramebuffer          = 0x8CA8
	glDrawFramebuffer          = 0x8CA9
	glColorAttachment0         = 0x8CE0
	glDepthAttachment          = 0x8D00
	glFramebuffer              = 0x8D40
	glCopyReadBuffer           = 0x8F36
	glCopyWriteBuffer          = 0x8F37
	glUniformBuffer            = 0x8A11
	glStaticDraw               = 0x88E4
	glDynamicDraw              = 0x88E8
	glFragmentShader           = 0x8B30
	glVertexShader             = 0x8B31
	glMax3DTextureSize         = 0x8073
	glMaxArrayTextureLayers    = 0x88FF
	glMaxVertexUniformBlocks   = 0x8A2B
	glMaxFragmentUniformBlocks = 0x8A2D
	glMaxUniformBlockSize      = 0x8A30
	glUniformBufferOffsetAlign = 0x8A34
	glMaxColorAttachments      = 0x8CDF
	glMaxDrawBuffers           = 0x8824
	glMaxVertexAttribs         = 0x8869
	glMaxVaryingComponents     = 0x8B4B
	glUnmaskedVendorWebGL      = 0x9245
	glUnmaskedRendererWebGL    = 0x9246
	glInvalidIndex             = 0xFFFFFFFF
	glColor                    = 0x1800
	glDepth                    = 0x1801
)

type glContext interface {
	getError() uint32
	isContextLost() bool
	getParameterInt(pname uint32) int
	getParameterString(pname uint32) string
	supportedExtensions() []string
	enableExtension(name string) bool
	createVertexArray() glObject
	bindVertexArray(vao glObject)
	deleteVertexArray(vao glObject)
	setCanvasSize(width, height uint32)
	onContextLost(callback func()) (release func())

	enable(capability uint32)
	disable(capability uint32)
	pixelStorei(pname uint32, param int)
	viewport(x, y, width, height int)
	scissor(x, y, width, height int)
	depthRange(near, far float32)
	frontFace(mode uint32)
	cullFace(mode uint32)
	depthFunc(function uint32)
	depthMask(enabled bool)
	colorMask(r, g, b, a bool)
	blendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha uint32)
	blendEquationSeparate(modeRGB, modeAlpha uint32)

	createBuffer() glObject
	deleteBuffer(buffer glObject)
	bindBuffer(target uint32, buffer glObject)
	bufferData(target uint32, size int, usage uint32)
	bufferSubData(target uint32, offset int, data []byte)
	bindBufferRange(target, index uint32, buffer glObject, offset, size int)

	createTexture() glObject
	deleteTexture(texture glObject)
	activeTexture(unit uint32)
	bindTexture(target uint32, texture glObject)
	texParameteri(target, pname uint32, param int)
	texStorage2D(target uint32, levels int, internalFormat uint32, width, height int)
	texSubImage2D(target uint32, level, x, y, width, height int, format, dataType uint32, data []byte)

	createSampler() glObject
	deleteSampler(sampler glObject)
	samplerParameteri(sampler glObject, pname uint32, param int)
	samplerParameterf(sampler glObject, pname uint32, param float32)
	bindSampler(unit uint32, sampler glObject)

	createShader(shaderType uint32) glObject
	shaderSource(shader glObject, source string)
	compileShader(shader glObject)
	getShaderCompileStatus(shader glObject) bool
	getShaderInfoLog(shader glObject) string
	deleteShader(shader glObject)
	createProgram() glObject
	attachShader(program, shader glObject)
	linkProgram(program glObject)
	getProgramLinkStatus(program glObject) bool
	getProgramInfoLog(program glObject) string
	deleteProgram(program glObject)
	useProgram(program glObject)
	getUniformBlockIndex(program glObject, name string) uint32
	uniformBlockBinding(program glObject, index, binding uint32)
	getUniformLocation(program glObject, name string) glObject
	uniform1i(location glObject, value int32)
	uniform1ui(location glObject, value uint32)

	createFramebuffer() glObject
	deleteFramebuffer(framebuffer glObject)
	bindFramebuffer(target uint32, framebuffer glObject)
	framebufferTexture2D(target, attachment, textarget uint32, texture glObject, level int)
	drawBuffers(buffers []uint32)
	invalidateFramebuffer(target uint32, attachments []uint32)
	clearBufferfv(buffer uint32, drawBuffer int, values [4]float32)
	blitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1 int, mask, filter uint32)
	readPixels(x, y, width, height int, format, dataType uint32, dst []byte)

	drawArrays(mode uint32, first, count int)
	drawArraysInstanced(mode uint32, first, count, instanceCount int)
}
