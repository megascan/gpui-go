//go:build darwin

package mtl

import (
	"testing"
	"unsafe"

	"github.com/megascan/gpui-go/internal/apple/objc"
)

func TestRegionLayout(t *testing.T) {
	if got := unsafe.Sizeof(Region{}); got != 6*unsafe.Sizeof(uint(0)) {
		t.Fatalf("sizeof(MTLRegion) = %d", got)
	}
	if got := unsafe.Sizeof(Viewport{}); got != 48 {
		t.Fatalf("sizeof(MTLViewport) = %d", got)
	}
	if got := unsafe.Sizeof(ClearColor{}); got != 32 {
		t.Fatalf("sizeof(MTLClearColor) = %d", got)
	}
}

func TestTextureRoundTrip(t *testing.T) {
	if err := Load(); err != nil {
		t.Skipf("Metal unavailable: %v", err)
	}
	device, ok := CreateSystemDefaultDevice()
	if !ok {
		t.Skip("no Metal device")
	}
	defer Release(device)
	objc.AutoreleasePool(func() {
		descriptor := NewTextureDescriptor()
		defer Release(descriptor)
		descriptor.SetWidth(4)
		descriptor.SetHeight(2)
		descriptor.SetPixelFormat(PixelFormatBGRA8Unorm)
		descriptor.SetUsage(TextureUsageShaderRead)
		if device.HasUnifiedMemory() {
			descriptor.SetStorageMode(StorageModeShared)
		} else {
			descriptor.SetStorageMode(StorageModeManaged)
		}
		texture := device.NewTextureWithDescriptor(descriptor)
		if texture == 0 {
			t.Fatal("newTextureWithDescriptor returned nil")
		}
		defer Release(texture)
		if texture.Width() != 4 || texture.Height() != 2 || texture.PixelFormat() != PixelFormatBGRA8Unorm {
			t.Fatalf("texture = %dx%d format %d", texture.Width(), texture.Height(), texture.PixelFormat())
		}
		region := Region{Size: Size{Width: 4, Height: 2, Depth: 1}}
		in := make([]byte, 4*2*4)
		for i := range in {
			in[i] = byte(i + 1)
		}
		texture.ReplaceRegion(region, 0, unsafe.Pointer(&in[0]), 16)
		out := make([]byte, len(in))
		texture.GetBytes(unsafe.Pointer(&out[0]), 16, region, 0)
		for i := range in {
			if in[i] != out[i] {
				t.Fatalf("byte %d = %d, want %d", i, out[i], in[i])
			}
		}
	})
}

func TestLibraryCompileError(t *testing.T) {
	if err := Load(); err != nil {
		t.Skipf("Metal unavailable: %v", err)
	}
	device, ok := CreateSystemDefaultDevice()
	if !ok {
		t.Skip("no Metal device")
	}
	defer Release(device)
	options := NewCompileOptions()
	defer Release(options)
	if _, err := device.NewLibraryWithSource("this is not metal", options); err == nil {
		t.Fatal("compiling invalid source succeeded")
	}
	library, err := device.NewLibraryWithSource("vertex float4 v() { return float4(0); }", options)
	if err != nil {
		t.Fatalf("compiling valid source: %v", err)
	}
	defer Release(library)
	f, ok := library.NewFunctionWithName("v")
	if !ok {
		t.Fatal("function v not found")
	}
	defer Release(f)
	if f.Name() != "v" {
		t.Fatalf("function name = %q", f.Name())
	}
	if _, ok := library.NewFunctionWithName("missing"); ok {
		t.Fatal("found a missing function")
	}
}
