//go:build linux

package wgpu

import (
	"crypto/sha256"
	"image"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func validateWGSL(t *testing.T, source string, features gpu.Features) {
	t.Helper()
	device, _ := testDeviceAndQueue(t, features)
	device.PushErrorScope(gpu.ErrorFilterValidation)
	module := device.CreateShaderModule(&gpu.ShaderModuleDescriptor{Label: "validate", WGSL: source})
	if err := device.PopErrorScope(); err != nil {
		t.Fatalf("shader should validate: %v", err)
	}
	module.Release()
}

func TestWgpuRenderer_WebglShaderIsValidWgslWithoutStorageBuffers(t *testing.T) {
	if strings.Contains(webglShaders, "var<storage") {
		t.Fatal("WebGL shaders must not use storage buffers")
	}
	validateWGSL(t, webglShaders, 0)
}

func TestWgpuRenderer_StorageBufferShaderIsValidWgsl(t *testing.T) {
	validateWGSL(t, storageBufferShaders, 0)
}

func TestWgpuRenderer_SubpixelStorageBufferShaderIsValidWgsl(t *testing.T) {
	validateWGSL(t, subpixelStorageBufferShaders, gpu.FeaturesDualSourceBlending)
}

func TestWgpuRenderer_SubpixelWebglShaderIsValidWgslWithoutStorageBuffers(t *testing.T) {
	if strings.Contains(subpixelWebglShaders, "var<storage") {
		t.Fatal("WebGL shaders must not use storage buffers")
	}
	validateWGSL(t, subpixelWebglShaders, gpu.FeaturesDualSourceBlending)
}

func TestShaders_MatchUpstream(t *testing.T) {
	refDir := filepath.Join("..", "..", "_ref", "gpui_wgpu", "src")
	if _, err := os.Stat(refDir); err != nil {
		t.Skipf("upstream reference absent: %v", err)
	}
	embedded := map[string]string{
		"shaders.wgsl":          shadersWGSL,
		"shaders_storage.wgsl":  shadersStorageWGSL,
		"shaders_webgl.wgsl":    shadersWebglWGSL,
		"shaders_subpixel.wgsl": shadersSubpixelWGSL,
	}
	for name, source := range embedded {
		ref, err := os.ReadFile(filepath.Join(refDir, name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if sha256.Sum256(ref) != sha256.Sum256([]byte(source)) {
			t.Errorf("%s differs from upstream", name)
		}
	}
}

func newTestHeadlessRenderer(t *testing.T) *WgpuHeadlessRenderer {
	t.Helper()
	renderer, err := NewWgpuHeadlessRenderer()
	if err != nil {
		if strings.Contains(err.Error(), "No usable headless GPU adapter found") {
			t.Skipf("no GPU adapter: %v", err)
		}
		t.Fatal(err)
	}
	t.Cleanup(renderer.Destroy)
	return renderer
}

func deviceSize(width, height int32) gpui.Size[gpui.DevicePixels] {
	return gpui.Size[gpui.DevicePixels]{Width: gpui.DevicePixels(width), Height: gpui.DevicePixels(height)}
}

func solidQuad(x, y, width, height float32, color gpui.Hsla) gpui.Quad {
	bounds := gpui.Bounds[gpui.ScaledPixels]{
		Origin: gpui.Point[gpui.ScaledPixels]{X: gpui.ScaledPixels(x), Y: gpui.ScaledPixels(y)},
		Size:   gpui.Size[gpui.ScaledPixels]{Width: gpui.ScaledPixels(width), Height: gpui.ScaledPixels(height)},
	}
	return gpui.Quad{
		Order:       0,
		BorderStyle: gpui.BorderStyleSolid,
		Bounds:      bounds,
		ContentMask: gpui.ContentMask[gpui.ScaledPixels]{Bounds: bounds},
		Background:  gpui.SolidBackground(color),
		BorderColor: color,
	}
}

func assertPixel(t *testing.T, img *image.RGBA, x, y int, expected [4]uint8) {
	t.Helper()
	p := img.PixOffset(x, y)
	actual := [4]uint8(img.Pix[p : p+4])
	for i := range 4 {
		diff := int(actual[i]) - int(expected[i])
		if diff < -3 || diff > 3 {
			t.Fatalf("pixel (%d, %d) was %v, expected %v", x, y, actual, expected)
		}
	}
}

var (
	red   = [4]uint8{255, 0, 0, 255}
	blue  = [4]uint8{0, 0, 255, 255}
	black = [4]uint8{0, 0, 0, 255}
)

func assertDimensions(t *testing.T, img *image.RGBA, width, height int) {
	t.Helper()
	if got := img.Bounds().Size(); got.X != width || got.Y != height {
		t.Fatalf("image dimensions were (%d, %d), expected (%d, %d)", got.X, got.Y, width, height)
	}
}

func TestWgpuRenderer_HeadlessRendererDrawsQuadsWithDistinctColors(t *testing.T) {
	renderer := newTestHeadlessRenderer(t)
	var scene gpui.Scene
	scene.InsertPrimitive(solidQuad(0, 0, 32, 32, gpui.Red()))
	scene.InsertPrimitive(solidQuad(32, 0, 32, 32, gpui.Blue()))
	scene.Finish()

	img, err := renderer.RenderSceneToImage(&scene, deviceSize(64, 32))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 64, 32)
	assertPixel(t, img, 8, 16, red)
	assertPixel(t, img, 24, 16, red)
	assertPixel(t, img, 40, 16, blue)
	assertPixel(t, img, 56, 16, blue)
}

func TestWgpuRenderer_HeadlessRendererCapturesEachRequestedSize(t *testing.T) {
	renderer := newTestHeadlessRenderer(t)
	var scene gpui.Scene
	scene.InsertPrimitive(solidQuad(2, 2, 4, 3, gpui.Red()))
	scene.Finish()

	img, err := renderer.RenderSceneToImage(&scene, deviceSize(13, 7))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 13, 7)
	assertPixel(t, img, 0, 0, black)
	assertPixel(t, img, 3, 3, red)
	assertPixel(t, img, 12, 6, black)

	img, err = renderer.RenderSceneToImage(&scene, deviceSize(17, 9))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 17, 9)
	assertPixel(t, img, 3, 3, red)
	assertPixel(t, img, 16, 8, black)

	var empty gpui.Scene
	img, err = renderer.RenderSceneToImage(&empty, deviceSize(13, 7))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 13, 7)
	for i := 0; i < len(img.Pix); i += 4 {
		if [4]uint8(img.Pix[i:i+4]) != black {
			t.Fatalf("pixel %d was %v, expected %v", i/4, img.Pix[i:i+4], black)
		}
	}
}

func TestWgpuRenderer_HeadlessRendererReusesTargetForSameSize(t *testing.T) {
	renderer := newTestHeadlessRenderer(t)
	targetTexture := func() gpu.Texture {
		if renderer.renderTarget == nil {
			return nil
		}
		return renderer.renderTarget.texture
	}

	var scene gpui.Scene
	if err := renderer.RenderScene(&scene, deviceSize(16, 16)); err != nil {
		t.Fatal(err)
	}
	first := targetTexture()
	if first == nil {
		t.Fatal("expected a render target")
	}

	if err := renderer.RenderScene(&scene, deviceSize(16, 16)); err != nil {
		t.Fatal(err)
	}
	if targetTexture() != first {
		t.Fatal("render target was recreated for the same size")
	}

	if err := renderer.RenderScene(&scene, deviceSize(16, 17)); err != nil {
		t.Fatal(err)
	}
	if targetTexture() == first {
		t.Fatal("render target was not recreated for a new size")
	}
}

func TestWgpuRenderer_HeadlessRendererRejectsInvalidSizes(t *testing.T) {
	renderer := newTestHeadlessRenderer(t)
	tooLarge := int32(renderer.core.maxTextureSize) + 1

	var scene gpui.Scene
	for _, size := range []gpui.Size[gpui.DevicePixels]{
		deviceSize(0, 8),
		deviceSize(8, 0),
		deviceSize(-1, 8),
		deviceSize(tooLarge, 8),
	} {
		if err := renderer.RenderScene(&scene, size); err == nil {
			t.Fatalf("%v should be rejected", size)
		}
		if _, err := renderer.RenderSceneToImage(&scene, size); err == nil {
			t.Fatalf("%v should be rejected", size)
		}
	}

	img, err := renderer.RenderSceneToImage(&scene, deviceSize(4, 4))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 4, 4)
}

func TestWgpuRenderer_HeadlessRendererDrawsOneSolidQuad(t *testing.T) {
	renderer := newTestHeadlessRenderer(t)
	var scene gpui.Scene
	scene.InsertPrimitive(solidQuad(4, 4, 8, 8, gpui.Blue()))
	scene.Finish()

	img, err := renderer.RenderSceneToImage(&scene, deviceSize(16, 16))
	if err != nil {
		t.Fatal(err)
	}
	assertDimensions(t, img, 16, 16)
	for y := range 16 {
		for x := range 16 {
			expected := black
			if x >= 4 && x < 12 && y >= 4 && y < 12 {
				expected = blue
			}
			assertPixel(t, img, x, y, expected)
		}
	}
}

func TestWgpuRenderer_WebglRecordSizesMatchShaderWordStrides(t *testing.T) {
	for _, c := range []struct {
		name  string
		size  uintptr
		words uintptr
	}{
		{"Quad", unsafe.Sizeof(gpui.Quad{}), 40},
		{"Shadow", unsafe.Sizeof(gpui.Shadow{}), 28},
		{"PathTile", unsafe.Sizeof(gpui.PathTile{}), 14},
		{"PathCurve", unsafe.Sizeof(gpui.PathCurve{}), 10},
		{"PathPaint", unsafe.Sizeof(gpui.PathPaint{}), 28},
		{"Underline", unsafe.Sizeof(gpui.Underline{}), 16},
		{"MonochromeSprite", unsafe.Sizeof(gpui.MonochromeSprite{}), 28},
		{"SubpixelSprite", unsafe.Sizeof(gpui.SubpixelSprite{}), 28},
		{"PolychromeSprite", unsafe.Sizeof(gpui.PolychromeSprite{}), 24},
	} {
		if c.size != c.words*4 {
			t.Errorf("size of %s is %d, expected %d", c.name, c.size, c.words*4)
		}
	}
}

func TestWgpuRenderer_WebglQuadLayoutMatchesFixedDecoder(t *testing.T) {
	quad := gpui.Quad{
		Order:       41,
		BorderStyle: gpui.BorderStyleDashed,
		Bounds: gpui.Bounds[gpui.ScaledPixels]{
			Origin: gpui.Point[gpui.ScaledPixels]{X: 1, Y: 2},
			Size:   gpui.Size[gpui.ScaledPixels]{Width: 3, Height: 4},
		},
		ContentMask: gpui.ContentMask[gpui.ScaledPixels]{Bounds: gpui.Bounds[gpui.ScaledPixels]{
			Origin: gpui.Point[gpui.ScaledPixels]{X: 5, Y: 6},
			Size:   gpui.Size[gpui.ScaledPixels]{Width: 7, Height: 8},
		}},
		Background: gpui.LinearGradient(
			11,
			gpui.NewLinearColorStop(gpui.Hsla{H: 12, S: 13, L: 14, A: 15}, 16),
			gpui.NewLinearColorStop(gpui.Hsla{H: 17, S: 18, L: 19, A: 20}, 21),
		).ColorSpace(gpui.ColorSpaceOklab),
		BorderColor:  gpui.Hsla{H: 22, S: 23, L: 24, A: 25},
		CornerRadii:  gpui.Corners[gpui.ScaledPixels]{TopLeft: 26, TopRight: 27, BottomRight: 28, BottomLeft: 29},
		BorderWidths: gpui.Edges[gpui.ScaledPixels]{Top: 30, Right: 31, Bottom: 32, Left: 33},
	}

	bytes := instanceBytes([]gpui.Quad{quad}).data
	words := unsafe.Slice((*uint32)(unsafe.Pointer(&bytes[0])), len(bytes)/4)
	f := math.Float32bits
	expected := []uint32{
		41, 1,
		f(1), f(2), f(3), f(4),
		f(5), f(6), f(7), f(8),
		1, 1, 0, 0, 0, 0,
		f(11),
		f(12), f(13), f(14), f(15), f(16),
		f(17), f(18), f(19), f(20), f(21),
		0,
		f(22), f(23), f(24), f(25),
		f(26), f(27), f(28), f(29),
		f(30), f(31), f(32), f(33),
	}
	if !slices.Equal(words, expected) {
		t.Fatalf("quad words were %v, expected %v", words, expected)
	}
}
