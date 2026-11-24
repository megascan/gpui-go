//go:build js && wasm

package webgl2

import (
	"fmt"
	"strings"
)

type fakeGL struct {
	calls        []string
	next         glObject
	params       map[uint32]int
	strings      map[uint32]string
	extensions   []string
	locations    map[glObject]string
	failCompile  bool
	lost         bool
	lostCallback func()
	released     int
	readFill     byte
}

func newFakeGL() *fakeGL {
	return &fakeGL{
		params: map[uint32]int{
			glMaxTextureSize:           4096,
			glUniformBufferOffsetAlign: 256,
			glMaxVertexUniformBlocks:   12,
			glMaxFragmentUniformBlocks: 12,
			glMaxUniformBlockSize:      65536,
			glMaxColorAttachments:      8,
			glMaxDrawBuffers:           8,
			glMaxVertexAttribs:         16,
		},
		strings:   map[uint32]string{glVendor: "WebKit", glRenderer: "WebKit WebGL", glVersion: "WebGL 2.0"},
		locations: map[glObject]string{},
	}
}

func (f *fakeGL) log(format string, args ...any) {
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}

func (f *fakeGL) reset() { f.calls = nil }

func (f *fakeGL) count(prefix string) int {
	n := 0
	for _, c := range f.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeGL) has(call string) bool {
	for _, c := range f.calls {
		if c == call {
			return true
		}
	}
	return false
}

func (f *fakeGL) obj() glObject { f.next++; return f.next }

func (f *fakeGL) getError() uint32                   { return glNoError }
func (f *fakeGL) isContextLost() bool                { return f.lost }
func (f *fakeGL) getParameterInt(p uint32) int       { return f.params[p] }
func (f *fakeGL) getParameterString(p uint32) string { return f.strings[p] }
func (f *fakeGL) supportedExtensions() []string      { return f.extensions }
func (f *fakeGL) setCanvasSize(w, h uint32)          { f.log("canvas %dx%d", w, h) }
func (f *fakeGL) onContextLost(cb func()) func() {
	f.lostCallback = cb
	return func() { f.released++; f.lostCallback = nil }
}
func (f *fakeGL) enable(c uint32)             { f.log("enable %#x", c) }
func (f *fakeGL) disable(c uint32)            { f.log("disable %#x", c) }
func (f *fakeGL) pixelStorei(p uint32, v int) { f.log("pixelStorei %#x %d", p, v) }
func (f *fakeGL) viewport(x, y, w, h int)     { f.log("viewport %d %d %d %d", x, y, w, h) }
func (f *fakeGL) scissor(x, y, w, h int)      { f.log("scissor %d %d %d %d", x, y, w, h) }
func (f *fakeGL) depthRange(n, fa float32)    { f.log("depthRange %g %g", n, fa) }
func (f *fakeGL) frontFace(m uint32)          { f.log("frontFace %#x", m) }
func (f *fakeGL) cullFace(m uint32)           { f.log("cullFace %#x", m) }
func (f *fakeGL) depthFunc(fn uint32)         { f.log("depthFunc %#x", fn) }
func (f *fakeGL) depthMask(e bool)            { f.log("depthMask %t", e) }
func (f *fakeGL) colorMask(r, g, b, a bool)   { f.log("colorMask %t %t %t %t", r, g, b, a) }
func (f *fakeGL) blendFuncSeparate(a, b, c, d uint32) {
	f.log("blendFuncSeparate %#x %#x %#x %#x", a, b, c, d)
}
func (f *fakeGL) blendEquationSeparate(a, b uint32) { f.log("blendEquationSeparate %#x %#x", a, b) }
func (f *fakeGL) createBuffer() glObject            { o := f.obj(); f.log("createBuffer %d", o); return o }
func (f *fakeGL) deleteBuffer(b glObject)           { f.log("deleteBuffer %d", b) }
func (f *fakeGL) bindBuffer(t uint32, b glObject)   { f.log("bindBuffer %#x %d", t, b) }
func (f *fakeGL) bufferData(t uint32, size int, u uint32) {
	f.log("bufferData %#x %d %#x", t, size, u)
}
func (f *fakeGL) bufferSubData(t uint32, off int, data []byte) {
	f.log("bufferSubData %#x %d %d", t, off, len(data))
}
func (f *fakeGL) bindBufferRange(t, i uint32, b glObject, off, size int) {
	f.log("bindBufferRange %#x %d %d %d %d", t, i, b, off, size)
}
func (f *fakeGL) createTexture() glObject          { o := f.obj(); f.log("createTexture %d", o); return o }
func (f *fakeGL) deleteTexture(t glObject)         { f.log("deleteTexture %d", t) }
func (f *fakeGL) activeTexture(u uint32)           { f.log("activeTexture %d", u-glTexture0) }
func (f *fakeGL) bindTexture(t uint32, x glObject) { f.log("bindTexture %#x %d", t, x) }
func (f *fakeGL) texParameteri(t, p uint32, v int) { f.log("texParameteri %#x %#x %#x", t, p, v) }
func (f *fakeGL) texStorage2D(t uint32, l int, i uint32, w, h int) {
	f.log("texStorage2D %#x %d %#x %d %d", t, l, i, w, h)
}
func (f *fakeGL) texSubImage2D(t uint32, l, x, y, w, h int, fo, ty uint32, data []byte) {
	f.log("texSubImage2D %d %d %d %d %#x %#x %v", x, y, w, h, fo, ty, data)
}
func (f *fakeGL) createSampler() glObject  { o := f.obj(); f.log("createSampler %d", o); return o }
func (f *fakeGL) deleteSampler(s glObject) { f.log("deleteSampler %d", s) }
func (f *fakeGL) samplerParameteri(s glObject, p uint32, v int) {
	f.log("samplerParameteri %d %#x %#x", s, p, v)
}
func (f *fakeGL) samplerParameterf(s glObject, p uint32, v float32) {
	f.log("samplerParameterf %d %#x %g", s, p, v)
}
func (f *fakeGL) bindSampler(u uint32, s glObject) { f.log("bindSampler %d %d", u, s) }
func (f *fakeGL) createShader(t uint32) glObject {
	o := f.obj()
	f.log("createShader %#x", t)
	return o
}
func (f *fakeGL) shaderSource(glObject, string)        {}
func (f *fakeGL) compileShader(glObject)               {}
func (f *fakeGL) getShaderCompileStatus(glObject) bool { return !f.failCompile }
func (f *fakeGL) getShaderInfoLog(glObject) string     { return "ERROR: 0:1: syntax error" }
func (f *fakeGL) deleteShader(s glObject)              { f.log("deleteShader %d", s) }
func (f *fakeGL) createProgram() glObject              { o := f.obj(); f.log("createProgram %d", o); return o }
func (f *fakeGL) attachShader(p, s glObject)           {}
func (f *fakeGL) linkProgram(p glObject)               { f.log("linkProgram %d", p) }
func (f *fakeGL) getProgramLinkStatus(glObject) bool   { return true }
func (f *fakeGL) getProgramInfoLog(glObject) string    { return "" }
func (f *fakeGL) deleteProgram(p glObject)             { f.log("deleteProgram %d", p) }
func (f *fakeGL) useProgram(p glObject)                { f.log("useProgram %d", p) }
func (f *fakeGL) getUniformBlockIndex(p glObject, name string) uint32 {
	f.log("getUniformBlockIndex %s", name)
	return uint32(len(name) % 4)
}
func (f *fakeGL) uniformBlockBinding(p glObject, i, b uint32) {
	f.log("uniformBlockBinding %d %d %d", p, i, b)
}
func (f *fakeGL) getUniformLocation(p glObject, name string) glObject {
	o := f.obj()
	f.locations[o] = name
	return o
}
func (f *fakeGL) uniform1i(l glObject, v int32)   { f.log("uniform1i %s %d", f.locationName(l), v) }
func (f *fakeGL) uniform1ui(l glObject, v uint32) { f.log("uniform1ui %s %d", f.locationName(l), v) }
func (f *fakeGL) locationName(l glObject) string {
	if name, ok := f.locations[l]; ok {
		return name
	}
	return "?"
}
func (f *fakeGL) createFramebuffer() glObject {
	o := f.obj()
	f.log("createFramebuffer %d", o)
	return o
}
func (f *fakeGL) deleteFramebuffer(fb glObject)         { f.log("deleteFramebuffer %d", fb) }
func (f *fakeGL) bindFramebuffer(t uint32, fb glObject) { f.log("bindFramebuffer %#x %d", t, fb) }
func (f *fakeGL) framebufferTexture2D(t, a, tt uint32, x glObject, l int) {
	f.log("framebufferTexture2D %#x %#x %d", t, a, x)
}
func (f *fakeGL) drawBuffers(b []uint32)                     { f.log("drawBuffers %#x", b) }
func (f *fakeGL) invalidateFramebuffer(t uint32, a []uint32) { f.log("invalidateFramebuffer %#x", a) }
func (f *fakeGL) clearBufferfv(b uint32, d int, v [4]float32) {
	f.log("clearBufferfv %#x %d %v", b, d, v)
}
func (f *fakeGL) blitFramebuffer(a, b, c, d, e, g, h, i int, m, fi uint32) {
	f.log("blitFramebuffer %d %d %d %d %d %d %d %d", a, b, c, d, e, g, h, i)
}
func (f *fakeGL) readPixels(x, y, w, h int, fo, ty uint32, dst []byte) {
	f.log("readPixels %d %d %d %d %d", x, y, w, h, len(dst))
	for i := range dst {
		dst[i] = f.readFill + byte(i)
	}
}
func (f *fakeGL) drawArrays(m uint32, first, count int) {
	f.log("drawArrays %#x %d %d", m, first, count)
}
func (f *fakeGL) drawArraysInstanced(m uint32, first, count, n int) {
	f.log("drawArraysInstanced %#x %d %d %d", m, first, count, n)
}

func (f *fakeGL) enableExtension(name string) bool { f.log("getExtension %s", name); return true }
func (f *fakeGL) createVertexArray() glObject {
	o := f.obj()
	f.log("createVertexArray %d", o)
	return o
}
func (f *fakeGL) bindVertexArray(v glObject)   { f.log("bindVertexArray %d", v) }
func (f *fakeGL) deleteVertexArray(v glObject) { f.log("deleteVertexArray %d", v) }
