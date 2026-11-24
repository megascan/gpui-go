//go:build js && wasm

package webgl2

import (
	"syscall/js"
)

type jsGL struct {
	gl      js.Value
	canvas  js.Value
	objects map[glObject]js.Value
	nextID  glObject
}

func newJSGL(canvas, gl js.Value) *jsGL {
	return &jsGL{gl: gl, canvas: canvas, objects: map[glObject]js.Value{}}
}

func (g *jsGL) put(v js.Value) glObject {
	if v.IsNull() || v.IsUndefined() {
		return 0
	}
	g.nextID++
	g.objects[g.nextID] = v
	return g.nextID
}

func (g *jsGL) get(o glObject) js.Value {
	if o == 0 {
		return js.Null()
	}
	if v, ok := g.objects[o]; ok {
		return v
	}
	return js.Null()
}

func (g *jsGL) drop(o glObject) js.Value {
	v := g.get(o)
	delete(g.objects, o)
	return v
}

func bytesToJS(data []byte) js.Value {
	array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(array, data)
	return array
}

func typedView(data []byte, dataType uint32) js.Value {
	array := bytesToJS(data)
	switch dataType {
	case glUnsignedInt:
		return js.Global().Get("Uint32Array").New(array.Get("buffer"), 0, len(data)/4)
	case glFloat:
		return js.Global().Get("Float32Array").New(array.Get("buffer"), 0, len(data)/4)
	}
	return array
}

func (g *jsGL) getError() uint32       { return uint32(g.gl.Call("getError").Int()) }
func (g *jsGL) isContextLost() bool    { return g.gl.Call("isContextLost").Bool() }
func (g *jsGL) enable(c uint32)        { g.gl.Call("enable", c) }
func (g *jsGL) disable(c uint32)       { g.gl.Call("disable", c) }
func (g *jsGL) frontFace(m uint32)     { g.gl.Call("frontFace", m) }
func (g *jsGL) cullFace(m uint32)      { g.gl.Call("cullFace", m) }
func (g *jsGL) depthFunc(f uint32)     { g.gl.Call("depthFunc", f) }
func (g *jsGL) depthMask(enabled bool) { g.gl.Call("depthMask", enabled) }

func (g *jsGL) getParameterInt(pname uint32) int {
	v := g.gl.Call("getParameter", pname)
	if v.Type() != js.TypeNumber {
		return 0
	}
	return v.Int()
}

func (g *jsGL) getParameterString(pname uint32) string {
	v := g.gl.Call("getParameter", pname)
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}

func (g *jsGL) supportedExtensions() []string {
	v := g.gl.Call("getSupportedExtensions")
	if !v.Truthy() {
		return nil
	}
	out := make([]string, v.Length())
	for i := range out {
		out[i] = v.Index(i).String()
	}
	return out
}

func (g *jsGL) setCanvasSize(width, height uint32) {
	g.canvas.Set("width", width)
	g.canvas.Set("height", height)
}

func (g *jsGL) onContextLost(callback func()) func() {
	if !g.canvas.Truthy() {
		return func() {}
	}
	listener := js.FuncOf(func(js.Value, []js.Value) any {
		callback()
		return nil
	})
	g.canvas.Call("addEventListener", "webglcontextlost", listener)
	return func() {
		g.canvas.Call("removeEventListener", "webglcontextlost", listener)
		listener.Release()
	}
}

func (g *jsGL) pixelStorei(pname uint32, param int) { g.gl.Call("pixelStorei", pname, param) }
func (g *jsGL) viewport(x, y, w, h int)             { g.gl.Call("viewport", x, y, w, h) }
func (g *jsGL) scissor(x, y, w, h int)              { g.gl.Call("scissor", x, y, w, h) }
func (g *jsGL) depthRange(n, f float32)             { g.gl.Call("depthRange", n, f) }
func (g *jsGL) colorMask(r, gr, b, a bool)          { g.gl.Call("colorMask", r, gr, b, a) }

func (g *jsGL) blendFuncSeparate(srcRGB, dstRGB, srcAlpha, dstAlpha uint32) {
	g.gl.Call("blendFuncSeparate", srcRGB, dstRGB, srcAlpha, dstAlpha)
}

func (g *jsGL) blendEquationSeparate(modeRGB, modeAlpha uint32) {
	g.gl.Call("blendEquationSeparate", modeRGB, modeAlpha)
}

func (g *jsGL) createBuffer() glObject          { return g.put(g.gl.Call("createBuffer")) }
func (g *jsGL) deleteBuffer(b glObject)         { g.gl.Call("deleteBuffer", g.drop(b)) }
func (g *jsGL) bindBuffer(t uint32, b glObject) { g.gl.Call("bindBuffer", t, g.get(b)) }

func (g *jsGL) bufferData(target uint32, size int, usage uint32) {
	g.gl.Call("bufferData", target, size, usage)
}

func (g *jsGL) bufferSubData(target uint32, offset int, data []byte) {
	g.gl.Call("bufferSubData", target, offset, bytesToJS(data))
}

func (g *jsGL) bindBufferRange(target, index uint32, buffer glObject, offset, size int) {
	g.gl.Call("bindBufferRange", target, index, g.get(buffer), offset, size)
}

func (g *jsGL) createTexture() glObject          { return g.put(g.gl.Call("createTexture")) }
func (g *jsGL) deleteTexture(t glObject)         { g.gl.Call("deleteTexture", g.drop(t)) }
func (g *jsGL) activeTexture(unit uint32)        { g.gl.Call("activeTexture", unit) }
func (g *jsGL) bindTexture(t uint32, x glObject) { g.gl.Call("bindTexture", t, g.get(x)) }

func (g *jsGL) texParameteri(target, pname uint32, param int) {
	g.gl.Call("texParameteri", target, pname, param)
}

func (g *jsGL) texStorage2D(target uint32, levels int, internalFormat uint32, width, height int) {
	g.gl.Call("texStorage2D", target, levels, internalFormat, width, height)
}

func (g *jsGL) texSubImage2D(target uint32, level, x, y, width, height int, format, dataType uint32, data []byte) {
	g.gl.Call("texSubImage2D", target, level, x, y, width, height, format, dataType, typedView(data, dataType))
}

func (g *jsGL) createSampler() glObject  { return g.put(g.gl.Call("createSampler")) }
func (g *jsGL) deleteSampler(s glObject) { g.gl.Call("deleteSampler", g.drop(s)) }

func (g *jsGL) samplerParameteri(s glObject, pname uint32, param int) {
	g.gl.Call("samplerParameteri", g.get(s), pname, param)
}

func (g *jsGL) samplerParameterf(s glObject, pname uint32, param float32) {
	g.gl.Call("samplerParameterf", g.get(s), pname, param)
}

func (g *jsGL) bindSampler(unit uint32, s glObject) { g.gl.Call("bindSampler", unit, g.get(s)) }

func (g *jsGL) createShader(t uint32) glObject { return g.put(g.gl.Call("createShader", t)) }
func (g *jsGL) shaderSource(s glObject, src string) {
	g.gl.Call("shaderSource", g.get(s), src)
}
func (g *jsGL) compileShader(s glObject) { g.gl.Call("compileShader", g.get(s)) }
func (g *jsGL) getShaderCompileStatus(s glObject) bool {
	return g.gl.Call("getShaderParameter", g.get(s), 0x8B81).Truthy()
}
func (g *jsGL) getShaderInfoLog(s glObject) string {
	v := g.gl.Call("getShaderInfoLog", g.get(s))
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}
func (g *jsGL) deleteShader(s glObject)    { g.gl.Call("deleteShader", g.drop(s)) }
func (g *jsGL) createProgram() glObject    { return g.put(g.gl.Call("createProgram")) }
func (g *jsGL) attachShader(p, s glObject) { g.gl.Call("attachShader", g.get(p), g.get(s)) }
func (g *jsGL) linkProgram(p glObject)     { g.gl.Call("linkProgram", g.get(p)) }
func (g *jsGL) getProgramLinkStatus(p glObject) bool {
	return g.gl.Call("getProgramParameter", g.get(p), 0x8B82).Truthy()
}
func (g *jsGL) getProgramInfoLog(p glObject) string {
	v := g.gl.Call("getProgramInfoLog", g.get(p))
	if v.Type() != js.TypeString {
		return ""
	}
	return v.String()
}
func (g *jsGL) deleteProgram(p glObject) { g.gl.Call("deleteProgram", g.drop(p)) }
func (g *jsGL) useProgram(p glObject)    { g.gl.Call("useProgram", g.get(p)) }

func (g *jsGL) getUniformBlockIndex(p glObject, name string) uint32 {
	return uint32(g.gl.Call("getUniformBlockIndex", g.get(p), name).Int())
}

func (g *jsGL) uniformBlockBinding(p glObject, index, binding uint32) {
	g.gl.Call("uniformBlockBinding", g.get(p), index, binding)
}

func (g *jsGL) getUniformLocation(p glObject, name string) glObject {
	return g.put(g.gl.Call("getUniformLocation", g.get(p), name))
}

func (g *jsGL) uniform1i(l glObject, v int32)   { g.gl.Call("uniform1i", g.get(l), v) }
func (g *jsGL) uniform1ui(l glObject, v uint32) { g.gl.Call("uniform1ui", g.get(l), v) }

func (g *jsGL) createFramebuffer() glObject  { return g.put(g.gl.Call("createFramebuffer")) }
func (g *jsGL) deleteFramebuffer(f glObject) { g.gl.Call("deleteFramebuffer", g.drop(f)) }
func (g *jsGL) bindFramebuffer(t uint32, f glObject) {
	g.gl.Call("bindFramebuffer", t, g.get(f))
}

func (g *jsGL) framebufferTexture2D(target, attachment, textarget uint32, texture glObject, level int) {
	g.gl.Call("framebufferTexture2D", target, attachment, textarget, g.get(texture), level)
}

func uintsToJS(values []uint32) js.Value {
	array := js.Global().Get("Array").New(len(values))
	for i, v := range values {
		array.SetIndex(i, v)
	}
	return array
}

func (g *jsGL) drawBuffers(buffers []uint32) { g.gl.Call("drawBuffers", uintsToJS(buffers)) }

func (g *jsGL) invalidateFramebuffer(target uint32, attachments []uint32) {
	g.gl.Call("invalidateFramebuffer", target, uintsToJS(attachments))
}

func (g *jsGL) clearBufferfv(buffer uint32, drawBuffer int, values [4]float32) {
	array := js.Global().Get("Float32Array").New(4)
	for i, v := range values {
		array.SetIndex(i, v)
	}
	g.gl.Call("clearBufferfv", buffer, drawBuffer, array)
}

func (g *jsGL) blitFramebuffer(srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1 int, mask, filter uint32) {
	g.gl.Call("blitFramebuffer", srcX0, srcY0, srcX1, srcY1, dstX0, dstY0, dstX1, dstY1, mask, filter)
}

func (g *jsGL) readPixels(x, y, width, height int, format, dataType uint32, dst []byte) {
	array := js.Global().Get("Uint8Array").New(len(dst))
	g.gl.Call("readPixels", x, y, width, height, format, dataType, array)
	js.CopyBytesToGo(dst, array)
}

func (g *jsGL) drawArrays(mode uint32, first, count int) {
	g.gl.Call("drawArrays", mode, first, count)
}

func (g *jsGL) drawArraysInstanced(mode uint32, first, count, instanceCount int) {
	g.gl.Call("drawArraysInstanced", mode, first, count, instanceCount)
}

func (g *jsGL) enableExtension(name string) bool {
	return g.gl.Call("getExtension", name).Truthy()
}

func (g *jsGL) createVertexArray() glObject    { return g.put(g.gl.Call("createVertexArray")) }
func (g *jsGL) bindVertexArray(vao glObject)   { g.gl.Call("bindVertexArray", g.get(vao)) }
func (g *jsGL) deleteVertexArray(vao glObject) { g.gl.Call("deleteVertexArray", g.drop(vao)) }
