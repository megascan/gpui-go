//go:build linux

package wgpu

import (
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func testDeviceAndQueue(t *testing.T, features gpu.Features) (gpu.Device, gpu.Queue) {
	t.Helper()
	instance := createInstance(gpu.InstanceDescriptor{Backends: gpu.BackendsAll})
	adapter, err := instance.RequestAdapter(&gpu.RequestAdapterOptions{
		PowerPreference: gpu.PowerPreferenceLowPower,
	})
	if err != nil || adapter == nil {
		t.Skipf("no GPU adapter: %v", err)
	}
	if !adapter.Features().Contains(features) {
		t.Skipf("adapter %q lacks features %#x", adapter.GetInfo().Name, uint64(features))
	}
	device, queue, err := adapter.RequestDevice(&gpu.DeviceDescriptor{
		Label:            "wgpu_atlas_test_device",
		RequiredFeatures: features,
		RequiredLimits:   gpu.DownlevelDefaults().UsingResolution(adapter.Limits()).UsingAlignment(adapter.Limits()),
	})
	if err != nil {
		t.Fatalf("failed to request device: %v", err)
	}
	return device, queue
}

func imageKey(id uint) gpui.AtlasKey {
	return gpui.AtlasKeyFromImage(gpui.RenderImageParams{ImageID: gpui.ImageID(id), FrameIndex: 0})
}

func insertTile(t *testing.T, atlas *WgpuAtlas, key gpui.AtlasKey, size gpui.Size[gpui.DevicePixels]) gpui.AtlasTile {
	t.Helper()
	byteCount := int(size.Width) * int(size.Height) * 4
	tile, ok, err := atlas.GetOrInsertWith(key, func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		return size, make([]byte, byteCount), true, nil
	})
	if err != nil {
		t.Fatalf("allocation should succeed: %v", err)
	}
	if !ok {
		t.Fatal("callback returns Some")
	}
	return tile
}

func TestWgpuAtlas_BeforeFrameSkipsUploadsForRemovedTexture(t *testing.T) {
	device, queue := testDeviceAndQueue(t, 0)

	atlas := NewWgpuAtlas(device, queue, gpu.TextureFormatBgra8Unorm)
	key := imageKey(1)
	size := gpui.Size[gpui.DevicePixels]{Width: 1, Height: 1}
	build := func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		return size, []byte{0, 0, 0, 255}, true, nil
	}

	if _, ok, err := atlas.GetOrInsertWith(key, build); err != nil || !ok {
		t.Fatalf("tile should be created: %v", err)
	}
	atlas.Remove(key)
	atlas.BeforeFrame()
}

func TestWgpuAtlas_RemoveDeallocatesTileSpaceForReuse(t *testing.T) {
	device, queue := testDeviceAndQueue(t, 0)
	atlas := NewWgpuAtlas(device, queue, gpu.TextureFormatBgra8Unorm)

	small := gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}
	big := gpui.Size[gpui.DevicePixels]{Width: 700, Height: 700}

	keeperKey := imageKey(1)
	bigKeyA := imageKey(2)
	bigKeyB := imageKey(3)

	keeperTile := insertTile(t, atlas, keeperKey, small)
	tileA := insertTile(t, atlas, bigKeyA, big)
	if keeperTile.TextureID != tileA.TextureID {
		t.Fatalf("keeper %v and big tile %v should share a texture", keeperTile.TextureID, tileA.TextureID)
	}

	atlas.Remove(bigKeyA)
	tileB := insertTile(t, atlas, bigKeyB, big)
	if tileB.TextureID != keeperTile.TextureID {
		t.Fatalf("reinserted big tile %v should reuse texture %v", tileB.TextureID, keeperTile.TextureID)
	}
}

func TestWgpuAtlas_ReusedTextureIDHasNewGeneration(t *testing.T) {
	device, queue := testDeviceAndQueue(t, 0)
	atlas := NewWgpuAtlas(device, queue, gpu.TextureFormatBgra8Unorm)
	size := gpui.Size[gpui.DevicePixels]{Width: 700, Height: 700}

	firstKey := imageKey(1)
	firstTile := insertTile(t, atlas, firstKey, size)
	firstInfo, ok := atlas.GetTextureInfo(firstTile.TextureID)
	if !ok {
		t.Fatal("first texture should exist")
	}
	atlas.Remove(firstKey)
	if _, ok := atlas.GetTextureInfo(firstTile.TextureID); ok {
		t.Fatal("removed texture should be gone")
	}

	secondTile := insertTile(t, atlas, imageKey(2), size)
	secondInfo, ok := atlas.GetTextureInfo(secondTile.TextureID)
	if !ok {
		t.Fatal("second texture should exist")
	}

	if secondTile.TextureID != firstTile.TextureID {
		t.Fatalf("texture id %v should be reused, got %v", firstTile.TextureID, secondTile.TextureID)
	}
	if secondInfo.Generation == firstInfo.Generation {
		t.Fatalf("generation %d should change", firstInfo.Generation)
	}
}

func TestWgpuAtlas_SwizzleUploadDataPreservesBgraUploads(t *testing.T) {
	input := []byte{0x10, 0x20, 0x30, 0x40}
	if got := swizzleUploadData(input, gpu.TextureFormatBgra8Unorm); !slices.Equal(got, input) {
		t.Fatalf("got %x, want %x", got, input)
	}
}

func TestWgpuAtlas_SwizzleUploadDataConvertsBgraToRgba(t *testing.T) {
	input := []byte{0x10, 0x20, 0x30, 0x40, 0xAA, 0xBB, 0xCC, 0xDD}
	want := []byte{0x30, 0x20, 0x10, 0x40, 0xCC, 0xBB, 0xAA, 0xDD}
	if got := swizzleUploadData(input, gpu.TextureFormatRgba8Unorm); !slices.Equal(got, want) {
		t.Fatalf("got %x, want %x", got, want)
	}
}

func atlasTextureCounts(atlas *WgpuAtlas, kind gpui.AtlasTextureKind) (int, int) {
	atlas.mu.Lock()
	defer atlas.mu.Unlock()
	list := atlas.state.Backend.storage.index(kind)
	return len(list.Textures), len(list.FreeList)
}

func TestWgpuAtlas_OversizedInsertDoesNotKeepTexture(t *testing.T) {
	device, queue := testDeviceAndQueue(t, 0)
	atlas := NewWgpuAtlas(device, queue, gpu.TextureFormatBgra8Unorm)
	max := gpui.DevicePixels(device.Limits().MaxTextureDimension2D)

	insertTile(t, atlas, imageKey(1), gpui.Size[gpui.DevicePixels]{Width: 16, Height: 16})
	beforeTextures, beforeFree := atlasTextureCounts(atlas, gpui.AtlasTextureKindPolychrome)

	for _, size := range []gpui.Size[gpui.DevicePixels]{{Width: max + 1, Height: 1}, {Width: 1, Height: max + 1}} {
		calls := 0
		_, _, err := atlas.GetOrInsertWith(imageKey(2), func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
			calls++
			return size, make([]byte, int(size.Width)*int(size.Height)*4), true, nil
		})
		if err == nil {
			t.Fatalf("oversized insert %v should fail", size)
		}
		if calls != 1 {
			t.Fatalf("build called %d times", calls)
		}
		textures, free := atlasTextureCounts(atlas, gpui.AtlasTextureKindPolychrome)
		if textures != beforeTextures || free != beforeFree {
			t.Fatalf("textures/free = %d/%d, want %d/%d", textures, free, beforeTextures, beforeFree)
		}
	}
}

func TestWgpuAtlas_ShortUploadDataIsRejected(t *testing.T) {
	device, queue := testDeviceAndQueue(t, 0)
	atlas := NewWgpuAtlas(device, queue, gpu.TextureFormatBgra8Unorm)
	size := gpui.Size[gpui.DevicePixels]{Width: 8, Height: 8}

	calls := 0
	build := func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		calls++
		return size, make([]byte, 8*8*4-1), true, nil
	}
	for range 2 {
		if _, _, err := atlas.GetOrInsertWith(imageKey(1), build); err == nil {
			t.Fatal("short upload data should fail")
		}
	}
	if calls != 2 {
		t.Fatalf("build called %d times, want 2 (failed tiles are not cached)", calls)
	}
	if textures, _ := atlasTextureCounts(atlas, gpui.AtlasTextureKindPolychrome); textures != 0 {
		t.Fatalf("%d textures allocated for rejected uploads", textures)
	}
	atlas.BeforeFrame()
}
