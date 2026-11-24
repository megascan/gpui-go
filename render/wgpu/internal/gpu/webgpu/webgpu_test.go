//go:build linux

package webgpu

import (
	"testing"

	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func testDevice(t *testing.T) (gpu.Device, gpu.Queue) {
	t.Helper()
	instance := CreateInstance(gpu.InstanceDescriptor{Backends: gpu.BackendsAll})
	adapter, err := instance.RequestAdapter(&gpu.RequestAdapterOptions{PowerPreference: gpu.PowerPreferenceLowPower})
	if err != nil || adapter == nil {
		t.Skipf("no GPU adapter: %v", err)
	}
	device, queue, err := adapter.RequestDevice(&gpu.DeviceDescriptor{
		Label:          "webgpu_test",
		RequiredLimits: gpu.DownlevelDefaults().UsingResolution(adapter.Limits()).UsingAlignment(adapter.Limits()),
	})
	if err != nil {
		t.Fatalf("RequestDevice: %v", err)
	}
	return device, queue
}

func TestDevice_ErrorScopeCapturesValidation(t *testing.T) {
	device, _ := testDevice(t)
	var uncaptured []string
	device.OnUncapturedError(func(err *gpu.Error) { uncaptured = append(uncaptured, err.Message) })

	device.PushErrorScope(gpu.ErrorFilterValidation)
	device.CreateTexture(&gpu.TextureDescriptor{
		Size:      gpu.Extent3d{Width: 0, Height: 4},
		Dimension: gpu.TextureDimensionD2,
		Format:    gpu.TextureFormatBgra8Unorm,
		Usage:     gpu.TextureUsagesTextureBinding,
	})
	if err := device.PopErrorScope(); err == nil {
		t.Fatal("zero-sized texture should be a validation error")
	}
	if len(uncaptured) != 0 {
		t.Fatalf("scoped error leaked: %v", uncaptured)
	}

	device.CreateTexture(&gpu.TextureDescriptor{
		Size:      gpu.Extent3d{Width: 0, Height: 4},
		Dimension: gpu.TextureDimensionD2,
		Format:    gpu.TextureFormatBgra8Unorm,
		Usage:     gpu.TextureUsagesTextureBinding,
	})
	if len(uncaptured) != 1 {
		t.Fatalf("unscoped error should reach the handler once, got %v", uncaptured)
	}

	device.PushErrorScope(gpu.ErrorFilterValidation)
	device.CreateTexture(&gpu.TextureDescriptor{
		Size:      gpu.Extent3d{Width: 4, Height: 4},
		Dimension: gpu.TextureDimensionD2,
		Format:    gpu.TextureFormatBgra8Unorm,
		Usage:     gpu.TextureUsagesRenderAttachment | gpu.TextureUsagesTransientAttachment,
	})
	if err := device.PopErrorScope(); err == nil {
		t.Fatal("TRANSIENT_ATTACHMENT should be reported, not passed to wgpu-native")
	}
}

func TestQueue_SubmitClearAndReadBack(t *testing.T) {
	device, queue := testDevice(t)
	var uncaptured []string
	device.OnUncapturedError(func(err *gpu.Error) { uncaptured = append(uncaptured, err.Message) })

	const width, height = 4, 4
	target := device.CreateTexture(&gpu.TextureDescriptor{
		Size:      gpu.Extent3d{Width: width, Height: height, DepthOrArrayLayers: 1},
		Dimension: gpu.TextureDimensionD2,
		Format:    gpu.TextureFormatRgba8Unorm,
		Usage:     gpu.TextureUsagesRenderAttachment | gpu.TextureUsagesCopySrc,
	})
	view := target.CreateView(nil)
	readback := device.CreateBuffer(&gpu.BufferDescriptor{
		Size:  gpu.CopyBytesPerRowAlignment * height,
		Usage: gpu.BufferUsagesMapRead | gpu.BufferUsagesCopyDst,
	})

	encoder := device.CreateCommandEncoder(nil)
	pass := encoder.BeginRenderPass(&gpu.RenderPassDescriptor{
		ColorAttachments: []gpu.RenderPassColorAttachment{{
			View:       view,
			LoadOp:     gpu.LoadOpClear,
			StoreOp:    gpu.StoreOpStore,
			ClearValue: gpu.Color{R: 1, A: 1},
		}},
	})
	pass.End()
	encoder.CopyTextureToBuffer(
		gpu.TexelCopyTextureInfo{Texture: target},
		gpu.TexelCopyBufferInfo{Buffer: readback, Layout: gpu.TexelCopyBufferLayout{BytesPerRow: gpu.CopyBytesPerRowAlignment}},
		gpu.Extent3d{Width: width, Height: height, DepthOrArrayLayers: 1},
	)
	queue.Submit(encoder.Finish())

	var mapErr error
	mapped := false
	readback.MapAsync(gpu.MapModeRead, 0, readback.Size(), func(err error) { mapErr, mapped = err, true })
	for !mapped {
		if err := device.Poll(true); err != nil {
			t.Fatal(err)
		}
	}
	if mapErr != nil {
		t.Fatal(mapErr)
	}
	data := readback.GetMappedRange(0, readback.Size())
	if got := data[:4]; got[0] != 255 || got[1] != 0 || got[2] != 0 || got[3] != 255 {
		t.Fatalf("pixel = %v, want red", got)
	}
	readback.Unmap()
	if len(uncaptured) != 0 {
		t.Fatalf("uncaptured errors: %v", uncaptured)
	}
}

func TestSurface_ValidateSurfaceConfiguration(t *testing.T) {
	caps := gpu.SurfaceCapabilities{
		Formats:      []gpu.TextureFormat{gpu.TextureFormatBgra8Unorm, gpu.TextureFormatBgra8UnormSrgb},
		PresentModes: []gpu.PresentMode{gpu.PresentModeFifo},
		AlphaModes:   []gpu.CompositeAlphaMode{gpu.CompositeAlphaModeOpaque},
		Usages:       gpu.TextureUsagesRenderAttachment,
	}
	valid := func() gpu.SurfaceConfiguration {
		return gpu.SurfaceConfiguration{
			Usage:       gpu.TextureUsagesRenderAttachment,
			Format:      gpu.TextureFormatBgra8Unorm,
			Width:       100,
			Height:      100,
			PresentMode: gpu.PresentModeFifo,
			AlphaMode:   gpu.CompositeAlphaModeOpaque,
		}
	}
	cases := []struct {
		name   string
		mutate func(*gpu.SurfaceConfiguration)
		ok     bool
	}{
		{"valid", func(*gpu.SurfaceConfiguration) {}, true},
		{"undefined present mode is fifo", func(c *gpu.SurfaceConfiguration) { c.PresentMode = 0 }, true},
		{"auto alpha", func(c *gpu.SurfaceConfiguration) { c.AlphaMode = gpu.CompositeAlphaModeAuto }, true},
		{"srgb view format", func(c *gpu.SurfaceConfiguration) {
			c.ViewFormats = []gpu.TextureFormat{gpu.TextureFormatBgra8UnormSrgb}
		}, true},
		{"incompatible view format", func(c *gpu.SurfaceConfiguration) { c.ViewFormats = []gpu.TextureFormat{gpu.TextureFormatRgba8Unorm} }, false},
		{"too large", func(c *gpu.SurfaceConfiguration) { c.Width = 4097 }, false},
		{"unsupported present mode", func(c *gpu.SurfaceConfiguration) { c.PresentMode = gpu.PresentModeMailbox }, false},
		{"unsupported format", func(c *gpu.SurfaceConfiguration) { c.Format = gpu.TextureFormatRgba8Unorm }, false},
		{"unsupported alpha mode", func(c *gpu.SurfaceConfiguration) { c.AlphaMode = gpu.CompositeAlphaModePreMultiplied }, false},
		{"zero area", func(c *gpu.SurfaceConfiguration) { c.Height = 0 }, false},
	}
	for _, tc := range cases {
		config := valid()
		tc.mutate(&config)
		err := validateSurfaceConfiguration(&config, caps, 4096)
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok=%v", tc.name, err, tc.ok)
		}
	}
}
