//go:build js && wasm

package webgl2

import (
	"errors"
	"fmt"
	"syscall/js"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

const srgbPresentVert = `#version 300 es
precision mediump float;
// A triangle that fills the whole screen
const vec2[3] TRIANGLE_POS = vec2[](
  vec2( 0.0, -3.0),
  vec2(-3.0,  1.0),
  vec2( 3.0,  1.0)
);
const vec2[3] TRIANGLE_UV = vec2[](
  vec2( 0.5,  1.),
  vec2( -1.0,  -1.0),
  vec2( 2.0,  -1.0)
);
out vec2 uv;
void main() {
  uv = TRIANGLE_UV[gl_VertexID];
  gl_Position = vec4(TRIANGLE_POS[gl_VertexID], 0.0, 1.0);
}`

const srgbPresentFrag = `#version 300 es
precision mediump float;
in vec2 uv;
uniform sampler2D present_texture;
out vec4 frag;
vec4 linear_to_srgb(vec4 linear) {
    vec3 color_linear = linear.rgb;
    vec3 selector = ceil(color_linear - 0.0031308); // 0 if under value, 1 if over
    vec3 under = 12.92 * color_linear;
    vec3 over = 1.055 * pow(color_linear, vec3(0.41666)) - 0.055;
    vec3 result = mix(under, over, selector);
    return vec4(result, linear.a);
}
void main() {
  frag = linear_to_srgb(texture(present_texture, uv));
}`

// BrowserWebGl2Supported reports whether the browser exposes WebGL2RenderingContext.
func BrowserWebGl2Supported() bool {
	return js.Global().Get("WebGL2RenderingContext").Truthy()
}

// ErrNotWebGl2Instance is returned by CreateCanvasSurface for an instance of another backend.
var ErrNotWebGl2Instance = errors.New("instance is not a WebGL2 instance")

// CreateCanvasSurface creates a surface that presents to an HTML canvas element through a
// WebGL2 context, as wgpu-hal's Instance::create_surface_from_canvas.
func CreateCanvasSurface(inst gpu.Instance, canvas js.Value) (gpu.Surface, error) {
	i, ok := inst.(*instance)
	if !ok {
		return nil, ErrNotWebGl2Instance
	}
	if !canvas.Truthy() {
		return nil, errors.New("canvas element is undefined")
	}
	return i.createSurfaceFromCanvas(canvas)
}

func (i *instance) createSurfaceFromCanvas(canvas js.Value) (s gpu.Surface, err error) {
	var context js.Value
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("canvas.getContext() threw exception %v", r)
			}
		}()
		options := js.Global().Get("Object").New()
		options.Set("antialias", false)
		context = canvas.Call("getContext", "webgl2", options)
	}()
	if err != nil {
		return nil, err
	}
	if context.IsNull() || context.IsUndefined() {
		return nil, errors.New("canvas.getContext() returned null; webgl2 not available or canvas already in use")
	}
	if current, ok := i.context.(*jsGL); ok && current.gl.Equal(context) {
		return i.newSurface(current), nil
	}
	return i.newSurface(newJSGL(canvas, context)), nil
}

func (i *instance) newSurface(gl glContext) *surface {
	i.context = gl
	return &surface{gl: gl}
}

type swapchain struct {
	width, height uint32
	format        gpu.TextureFormat
	formatDesc    textureFormatDesc
	framebuffer   glObject
}

type surface struct {
	gl                 glContext
	srgbPresentProgram glObject
	swapchain          *swapchain
	texture            glObject
}

func (s *surface) GetCapabilities(a gpu.Adapter) gpu.SurfaceCapabilities {
	ad, ok := a.(*adapter)
	if !ok || ad.gl != s.gl {
		return gpu.SurfaceCapabilities{}
	}
	return gpu.SurfaceCapabilities{
		Formats:      []gpu.TextureFormat{gpu.TextureFormatRgba8Unorm, gpu.TextureFormatRgba8UnormSrgb},
		PresentModes: []gpu.PresentMode{gpu.PresentModeFifo},
		AlphaModes:   []gpu.CompositeAlphaMode{gpu.CompositeAlphaModeOpaque},
		Usages:       gpu.TextureUsagesRenderAttachment,
	}
}

func (s *surface) Configure(d gpu.Device, config *gpu.SurfaceConfiguration) {
	dev, ok := d.(*device)
	if !ok {
		return
	}
	desc, ok := describeTextureFormat(config.Format)
	if !ok || desc.depth {
		dev.report(gpu.ErrorFilterValidation, fmt.Sprintf("Surface does not support the format %s", config.Format))
		return
	}
	if config.Width == 0 || config.Height == 0 {
		dev.report(gpu.ErrorFilterValidation, fmt.Sprintf("Surface size %dx%d is invalid", config.Width, config.Height))
		return
	}
	gl := s.gl
	gl.setCanvasSize(config.Width, config.Height)
	if s.swapchain != nil {
		gl.deleteFramebuffer(s.swapchain.framebuffer)
		s.swapchain = nil
	}
	if s.srgbPresentProgram == 0 && config.Format.HasSrgbSuffix() {
		s.srgbPresentProgram = createSrgbPresentProgram(gl)
	}
	if s.texture != 0 {
		gl.deleteTexture(s.texture)
	}
	s.texture = gl.createTexture()
	gl.bindTexture(glTexture2D, s.texture)
	gl.texParameteri(glTexture2D, glTextureMinFilter, glNearest)
	gl.texParameteri(glTexture2D, glTextureMagFilter, glNearest)
	gl.texStorage2D(glTexture2D, 1, desc.internal, int(config.Width), int(config.Height))
	framebuffer := gl.createFramebuffer()
	gl.bindFramebuffer(glReadFramebuffer, framebuffer)
	gl.framebufferTexture2D(glReadFramebuffer, glColorAttachment0, glTexture2D, s.texture, 0)
	gl.bindTexture(glTexture2D, 0)
	s.swapchain = &swapchain{
		width:       config.Width,
		height:      config.Height,
		format:      config.Format,
		formatDesc:  desc,
		framebuffer: framebuffer,
	}
}

func createSrgbPresentProgram(gl glContext) glObject {
	program := gl.createProgram()
	vertex := gl.createShader(glVertexShader)
	gl.shaderSource(vertex, srgbPresentVert)
	gl.compileShader(vertex)
	fragment := gl.createShader(glFragmentShader)
	gl.shaderSource(fragment, srgbPresentFrag)
	gl.compileShader(fragment)
	gl.attachShader(program, vertex)
	gl.attachShader(program, fragment)
	gl.linkProgram(program)
	gl.deleteShader(vertex)
	gl.deleteShader(fragment)
	gl.bindTexture(glTexture2D, 0)
	return program
}

func (s *surface) GetCurrentTexture() gpu.CurrentSurfaceTexture {
	sc := s.swapchain
	if sc == nil {
		return gpu.CurrentSurfaceTexture{Status: gpu.CurrentSurfaceTextureValidation}
	}
	return gpu.CurrentSurfaceTexture{
		Status: gpu.CurrentSurfaceTextureSuccess,
		Texture: &texture{
			gl:           s.gl,
			raw:          s.texture,
			width:        sc.width,
			height:       sc.height,
			format:       sc.format,
			formatDesc:   sc.formatDesc,
			surfaceOwned: true,
		},
	}
}

func (s *surface) Present() {
	sc := s.swapchain
	if sc == nil {
		return
	}
	gl := s.gl
	if sc.format.HasSrgbSuffix() {
		gl.viewport(0, 0, int(sc.width), int(sc.height))
		gl.bindFramebuffer(glDrawFramebuffer, 0)
		gl.bindSampler(0, 0)
		gl.activeTexture(glTexture0)
		gl.bindTexture(glTexture2D, s.texture)
		gl.useProgram(s.srgbPresentProgram)
		gl.disable(glDepthTest)
		gl.disable(glStencilTest)
		gl.disable(glScissorTest)
		gl.disable(glBlend)
		gl.disable(glCullFace)
		gl.drawBuffers([]uint32{glBack})
		gl.drawArrays(glTriangles, 0, 3)
		return
	}
	gl.bindFramebuffer(glReadFramebuffer, sc.framebuffer)
	gl.bindFramebuffer(glDrawFramebuffer, 0)
	gl.blitFramebuffer(
		0, int(sc.height), int(sc.width), 0,
		0, 0, int(sc.width), int(sc.height),
		glColorBufferBit, glNearest,
	)
}

func (s *surface) Release() {
	gl := s.gl
	if s.swapchain != nil {
		gl.deleteFramebuffer(s.swapchain.framebuffer)
		s.swapchain = nil
	}
	if s.texture != 0 {
		gl.deleteTexture(s.texture)
		s.texture = 0
	}
	if s.srgbPresentProgram != 0 {
		gl.deleteProgram(s.srgbPresentProgram)
		s.srgbPresentProgram = 0
	}
}
