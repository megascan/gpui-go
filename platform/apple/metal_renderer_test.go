//go:build darwin

package apple

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"unsafe"

	"github.com/ebitengine/purego"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/cf"
	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/apple/objc"
)

func skipWithoutMetal(t *testing.T) {
	t.Helper()
	if err := mtl.Load(); err != nil {
		t.Skipf("Metal unavailable: %v", err)
	}
	device, ok := mtl.CreateSystemDefaultDevice()
	if !ok {
		t.Skip("no Metal device")
	}
	mtl.Release(device)
}

func TestMetalRenderer_RenderToImageRejectsZeroSizedLayer(t *testing.T) {
	skipWithoutMetal(t)
	objc.AutoreleasePool(func() {
		renderer, err := NewMetalRenderer(NewInstanceBufferPool(), false, false)
		if err != nil {
			t.Fatal(err)
		}
		defer renderer.Destroy()
		for _, s := range [][2]int32{{0, 0}, {0, 64}, {64, 0}} {
			renderer.UpdateDrawableSize(gpui.Size[gpui.DevicePixels]{Width: gpui.DevicePixels(s[0]), Height: gpui.DevicePixels(s[1])})
			if _, err := renderer.RenderToImage(new(gpui.Scene)); err == nil {
				t.Errorf("RenderToImage with drawable size %v succeeded", s)
			}
		}
	})
}

func TestMetalRenderer_DisplayLinkModeConsumesAndClearsDrawables(t *testing.T) {
	skipWithoutMetal(t)
	objc.AutoreleasePool(func() {
		renderer, err := NewMetalRenderer(NewInstanceBufferPool(), false, false)
		if err != nil {
			t.Fatal(err)
		}
		defer renderer.Destroy()
		renderer.UpdateDrawableSize(gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64})
		layer, ok := renderer.Layer()
		if !ok {
			t.Fatal("window renderer must supply a drawable")
		}
		next, ok := layer.NextDrawable()
		if !ok {
			t.Fatal("window renderer must supply a drawable")
		}
		drawable := mtl.Retain(next)
		defer mtl.Release(drawable)

		renderer.SetDisplayLinkAttached(true)
		if m := renderer.displayLinkMode; !m.attached || m.drawable != 0 {
			t.Fatalf("display link mode = %+v, want attached without drawable", m)
		}
		if old := renderer.SetFrameDrawable(mtl.Retain(drawable)); old != 0 {
			t.Fatal("SetFrameDrawable replaced a drawable")
		}
		if m := renderer.displayLinkMode; !m.attached || m.drawable == 0 {
			t.Fatalf("display link mode = %+v, want attached with drawable", m)
		}
		renderer.Draw(new(gpui.Scene))
		if m := renderer.displayLinkMode; !m.attached || m.drawable != 0 {
			t.Fatalf("display link mode = %+v, want attached without drawable", m)
		}
		if old := renderer.SetFrameDrawable(mtl.Retain(drawable)); old != 0 {
			t.Fatal("a frame that drew must consume its drawable")
		}
		old := renderer.SetFrameDrawable(0)
		if old == 0 {
			t.Fatal("a frame that did not draw must hand its drawable back")
		}
		mtl.Release(old)

		renderer.SetFrameDrawable(mtl.Retain(drawable))
		renderer.SetDisplayLinkAttached(false)
		if m := renderer.displayLinkMode; m.attached || m.drawable != 0 {
			t.Fatalf("display link mode = %+v, want detached", m)
		}
	})
}

var (
	cvPixelBufferCreate                  func(allocator uintptr, width, height uint, pixelFormatType uint32, attributes uintptr, out *uintptr) int32
	cvPixelBufferLockBaseAddress         func(pixelBuffer uintptr, flags uint64) int32
	cvPixelBufferUnlockBaseAddress       func(pixelBuffer uintptr, flags uint64) int32
	cvPixelBufferGetBaseAddressOfPlane   func(pixelBuffer uintptr, plane uint) unsafe.Pointer
	cvPixelBufferGetBytesPerRowOfPlane   func(pixelBuffer uintptr, plane uint) uint
	kCVPixelBufferIOSurfacePropertiesKey cf.TypeRef
)

func loadCoreVideoTestFuncs(t *testing.T) {
	t.Helper()
	if err := loadCoreVideo(); err != nil {
		t.Skipf("CoreVideo unavailable: %v", err)
	}
	purego.RegisterLibFunc(&cvPixelBufferCreate, coreVideoLib, "CVPixelBufferCreate")
	purego.RegisterLibFunc(&cvPixelBufferLockBaseAddress, coreVideoLib, "CVPixelBufferLockBaseAddress")
	purego.RegisterLibFunc(&cvPixelBufferUnlockBaseAddress, coreVideoLib, "CVPixelBufferUnlockBaseAddress")
	purego.RegisterLibFunc(&cvPixelBufferGetBaseAddressOfPlane, coreVideoLib, "CVPixelBufferGetBaseAddressOfPlane")
	purego.RegisterLibFunc(&cvPixelBufferGetBytesPerRowOfPlane, coreVideoLib, "CVPixelBufferGetBytesPerRowOfPlane")
	key, err := cf.Constant(coreVideoLib, "kCVPixelBufferIOSurfacePropertiesKey")
	if err != nil {
		t.Fatal(err)
	}
	kCVPixelBufferIOSurfacePropertiesKey = key
}

func TestMetalRenderer_RenderCoreVideoSurface(t *testing.T) {
	skipWithoutMetal(t)
	loadCoreVideoTestFuncs(t)
	objc.AutoreleasePool(func() {
		empty := cf.DictionaryCreate(nil, nil)
		attributes := cf.DictionaryCreate([]cf.TypeRef{kCVPixelBufferIOSurfacePropertiesKey}, []cf.TypeRef{empty.TypeRef()})
		empty.Release()
		var imageBuffer uintptr
		status := cvPixelBufferCreate(0, 16, 16, kCVPixelFormatType_420YpCbCr8BiPlanarFullRange, uintptr(attributes.Ref()), &imageBuffer)
		attributes.Release()
		if status != kCVReturnSuccess || imageBuffer == 0 {
			t.Fatalf("CVPixelBufferCreate status %d", status)
		}
		defer cf.Release(cf.TypeRef(imageBuffer))
		if s := cvPixelBufferLockBaseAddress(imageBuffer, 0); s != kCVReturnSuccess {
			t.Fatalf("CVPixelBufferLockBaseAddress status %d", s)
		}
		for plane := uint(0); plane < 2; plane++ {
			base := cvPixelBufferGetBaseAddressOfPlane(imageBuffer, plane)
			length := cvPixelBufferGetBytesPerRowOfPlane(imageBuffer, plane) * cvPixelBufferGetHeightOfPlane(imageBuffer, plane)
			planeBytes := unsafe.Slice((*byte)(base), length)
			for i := range planeBytes {
				planeBytes[i] = 128
			}
		}
		if s := cvPixelBufferUnlockBaseAddress(imageBuffer, 0); s != kCVReturnSuccess {
			t.Fatalf("CVPixelBufferUnlockBaseAddress status %d", s)
		}

		bounds := gpui.Bounds[gpui.ScaledPixels]{
			Size: gpui.Size[gpui.ScaledPixels]{Width: 16, Height: 16},
		}
		scene := new(gpui.Scene)
		scene.InsertPrimitive(gpui.PaintSurface{
			Order:       0,
			Bounds:      bounds,
			ContentMask: gpui.ContentMask[gpui.ScaledPixels]{Bounds: bounds},
			ImageBuffer: imageBuffer,
		})
		scene.Finish()

		renderer, err := NewMetalRendererHeadless(NewInstanceBufferPool())
		if err != nil {
			t.Fatal(err)
		}
		size := gpui.Size[gpui.DevicePixels]{Width: 16, Height: 16}
		img, err := renderer.RenderSceneToImage(scene, size)
		if err != nil {
			t.Fatal(err)
		}
		absDiff := func(a, b uint8) uint8 {
			if a > b {
				return a - b
			}
			return b - a
		}
		for i := 0; i < len(img.Pix); i += 4 {
			red, green, blue, alpha := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
			if absDiff(red, 128) > 2 || absDiff(green, 128) > 2 || absDiff(blue, 128) > 2 || alpha != 255 {
				t.Fatalf("pixel %d = %v, want about (128, 128, 128, 255)", i/4, img.Pix[i:i+4])
			}
		}

		descriptor := mtl.NewTextureDescriptor()
		defer mtl.Release(descriptor)
		descriptor.SetWidth(16)
		descriptor.SetHeight(16)
		descriptor.SetPixelFormat(mtl.PixelFormatBGRA8Unorm)
		descriptor.SetUsage(mtl.TextureUsageRenderTarget)
		usesSharedStorage := runtime.GOOS == "ios" || renderer.isAppleGPU
		if usesSharedStorage {
			descriptor.SetStorageMode(mtl.StorageModeShared)
		} else {
			descriptor.SetStorageMode(mtl.StorageModeManaged)
		}
		targetTexture := renderer.device.NewTextureWithDescriptor(descriptor)
		if targetTexture == 0 {
			t.Fatal("allocating target texture")
		}
		defer mtl.Release(targetTexture)
		commandBuffer, err := renderer.renderFrame(scene, targetTexture, size)
		if err != nil {
			t.Fatal(err)
		}
		defer mtl.Release(commandBuffer)
		if !usesSharedStorage {
			blit := commandBuffer.BlitCommandEncoder()
			blit.SynchronizeResource(targetTexture.ID())
			blit.EndEncoding()
		}
		scene.Clear()
		renderer.Destroy()
		commandBuffer.Commit()
		commandBuffer.WaitUntilCompleted()
		if got := commandBuffer.Status(); got != mtl.CommandBufferStatusCompleted {
			t.Fatalf("command buffer status = %d, want completed", got)
		}
		got, err := readTextureToImage(targetTexture)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Pix, img.Pix) || got.Rect != img.Rect {
			t.Fatal("render_frame output differs from render_scene_to_image")
		}
	})
}

func TestMetalRenderer_InputIndicesMatchSceneHeader(t *testing.T) {
	for name, value := range map[string]int{
		"GlobalInputIndex_ViewportSize":     globalInputIndexViewportSize,
		"GlobalInputIndex_DepthScale":       globalInputIndexDepthScale,
		"ShadowInputIndex_Shadows":          shadowInputIndexShadows,
		"QuadInputIndex_Quads":              quadInputIndexQuads,
		"QuadInputIndex_QuadCount":          quadInputIndexQuadCount,
		"UnderlineInputIndex_Underlines":    underlineInputIndexUnderlines,
		"SpriteInputIndex_Sprites":          spriteInputIndexSprites,
		"SpriteInputIndex_AtlasTextureSize": spriteInputIndexAtlasTextureSize,
		"SpriteInputIndex_AtlasTexture":     spriteInputIndexAtlasTexture,
		"SurfaceInputIndex_Surfaces":        surfaceInputIndexSurfaces,
		"SurfaceInputIndex_TextureSize":     surfaceInputIndexTextureSize,
		"SurfaceInputIndex_YTexture":        surfaceInputIndexYTexture,
		"SurfaceInputIndex_CbCrTexture":     surfaceInputIndexCbCrTexture,
		"PathTileInputIndex_Tiles":          pathTileInputIndexTiles,
		"PathTileInputIndex_Curves":         pathTileInputIndexCurves,
		"PathTileInputIndex_Paints":         pathTileInputIndexPaints,
	} {
		if want := fmt.Sprintf("  %s = %d,\n", name, value); !strings.Contains(sceneHeader, want) {
			t.Errorf("scene.h does not declare %q", strings.TrimSpace(want))
		}
	}
	var b SurfaceBounds
	if got := unsafe.Sizeof(b); got != 32 {
		t.Errorf("sizeof(SurfaceBounds) = %d, want 32", got)
	}
	if got := unsafe.Offsetof(b.ContentMask); got != 16 {
		t.Errorf("offsetof(SurfaceBounds.content_mask) = %d, want 16", got)
	}
	if !strings.Contains(sceneHeader, "typedef struct SurfaceBounds {\n  struct Bounds_ScaledPixels bounds;\n  struct ContentMask_ScaledPixels content_mask;\n} SurfaceBounds;") {
		t.Error("scene.h SurfaceBounds differs from the Go struct")
	}
}
