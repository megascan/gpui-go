//go:build darwin

package apple

import (
	"bytes"
	"math"
	"runtime"
	"testing"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/apple/mtl"
)

func TestMetalAtlas_RemoveClearsStaleKeysFromTilesByKey(t *testing.T) {
	atlas := createAtlas(t)
	small := gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}

	keyA := makeImageKey(1, 0)
	keyB := makeImageKey(2, 0)
	keyC := makeImageKey(3, 0)

	tileA := insertTile(t, atlas, keyA, small)
	tileB := insertTile(t, atlas, keyB, small)
	tileC := insertTile(t, atlas, keyC, small)

	if tileA.TextureID != tileB.TextureID || tileB.TextureID != tileC.TextureID {
		t.Fatalf("texture ids %v %v %v differ", tileA.TextureID, tileB.TextureID, tileC.TextureID)
	}

	atlas.Remove(keyA)
	atlas.Remove(keyB)
	atlas.Remove(keyC)

	tileA2 := insertTile(t, atlas, keyA, small)
	if tileA2.TextureID != tileA.TextureID {
		t.Fatalf("re-inserted texture id = %v, want %v", tileA2.TextureID, tileA.TextureID)
	}
	texture, ok := atlas.MetalTexture(tileA2.TextureID)
	if !ok {
		t.Fatal("metal_texture(tile_a2.texture_id) is None")
	}
	mtl.Release(texture)
}

func TestMetalAtlas_MetalTextureIsNoneAfterLastTileRemoved(t *testing.T) {
	atlas := createAtlas(t)
	key := makeImageKey(1, 0)
	tile := insertTile(t, atlas, key, gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64})
	texture, ok := atlas.MetalTexture(tile.TextureID)
	if !ok {
		t.Fatal("metal_texture is None before removal")
	}
	mtl.Release(texture)

	atlas.Remove(key)
	if _, ok := atlas.MetalTexture(tile.TextureID); ok {
		t.Fatal("metal_texture is Some after the last tile was removed")
	}
}

func TestMetalAtlas_RemoveDeallocatesTileSpaceForReuse(t *testing.T) {
	atlas := createAtlas(t)
	small := gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}
	big := gpui.Size[gpui.DevicePixels]{Width: 700, Height: 700}

	keeperKey := makeImageKey(1, 0)
	bigKeyA := makeImageKey(2, 0)
	bigKeyB := makeImageKey(3, 0)

	keeperTile := insertTile(t, atlas, keeperKey, small)
	tileA := insertTile(t, atlas, bigKeyA, big)
	if keeperTile.TextureID != tileA.TextureID {
		t.Fatalf("big tile texture = %v, want %v", tileA.TextureID, keeperTile.TextureID)
	}

	atlas.Remove(bigKeyA)
	tileB := insertTile(t, atlas, bigKeyB, big)
	if tileB.TextureID != keeperTile.TextureID {
		t.Fatalf("reused tile texture = %v, want %v", tileB.TextureID, keeperTile.TextureID)
	}
}

func TestMetalAtlas_RemoveNonexistentKeyIsNoop(t *testing.T) {
	atlas := createAtlas(t)
	atlas.Remove(makeImageKey(999, 0))
}

func TestMetalAtlas_InvalidUploadDoesNotAllocate(t *testing.T) {
	atlas := createAtlas(t)
	atlas.mu.Lock()
	defer atlas.mu.Unlock()
	state := atlas.state
	for _, outer := range []struct {
		kind       gpui.AtlasTextureKind
		shortBytes []byte
		shortError string
	}{
		{gpui.AtlasTextureKindMonochrome, []byte{}, "atlas upload byte length is 0, expected at least 1"},
		{gpui.AtlasTextureKindPolychrome, []byte{0}, "atlas upload byte length is 1, expected at least 4"},
	} {
		for _, inner := range []struct {
			width, height int32
			bytes         []byte
			expectedError string
		}{
			{1, 1, outer.shortBytes, outer.shortError},
			{0, 1, []byte{}, "atlas dimensions must be positive"},
			{1, 0, []byte{}, "atlas dimensions must be positive"},
			{math.MinInt32, 1, []byte{}, "atlas dimensions must be positive"},
			{1, math.MinInt32, []byte{}, "atlas dimensions must be positive"},
		} {
			_, err := state.Backend.Insert(outer.kind, gpui.Size[gpui.DevicePixels]{
				Width:  gpui.DevicePixels(inner.width),
				Height: gpui.DevicePixels(inner.height),
			}, inner.bytes)
			if err == nil || err.Error() != inner.expectedError {
				t.Fatalf("insert(%v, %dx%d) error = %v, want %q", outer.kind, inner.width, inner.height, err, inner.expectedError)
			}
			for _, textures := range []*gpui.AtlasTextureList[*metalAtlasTexture]{
				&state.Backend.monochromeTextures,
				&state.Backend.polychromeTextures,
			} {
				if len(textures.Textures) != 0 || len(textures.FreeList) != 0 {
					t.Fatalf("(textures, free_list) = (%d, %d), want (0, 0)", len(textures.Textures), len(textures.FreeList))
				}
			}
		}
	}
}

func TestMetalAtlas_TextureUploadPreservesPixelLayout(t *testing.T) {
	atlas := createAtlas(t)
	atlas.mu.Lock()
	defer atlas.mu.Unlock()
	state := atlas.state
	for _, c := range []struct {
		kind          gpui.AtlasTextureKind
		bytesPerPixel int
	}{
		{gpui.AtlasTextureKindMonochrome, 1},
		{gpui.AtlasTextureKindPolychrome, 4},
	} {
		texture, err := state.Backend.pushTexture(gpui.Size[gpui.DevicePixels]{Width: 8, Height: 8}, c.kind)
		if err != nil {
			t.Fatal(err)
		}
		upload, err := newAtlasUpload(c.kind, gpui.Size[gpui.DevicePixels]{Width: 8, Height: 8}, make([]byte, 8*8*c.bytesPerPixel))
		if err != nil {
			t.Fatal(err)
		}
		if err := texture.upload(gpui.Point[gpui.DevicePixels]{}, &upload); err != nil {
			t.Fatal(err)
		}
		pixels := make([]byte, 3*2*c.bytesPerPixel)
		for i := range pixels {
			pixels[i] = byte(i + 1)
		}
		upload, err = newAtlasUpload(c.kind, gpui.Size[gpui.DevicePixels]{Width: 3, Height: 2}, pixels)
		if err != nil {
			t.Fatal(err)
		}
		if err := texture.upload(gpui.Point[gpui.DevicePixels]{X: 2, Y: 3}, &upload); err != nil {
			t.Fatal(err)
		}
		actual := make([]byte, 8*8*c.bytesPerPixel)
		texture.metalTexture.GetBytes(unsafe.Pointer(&actual[0]), uint(8*c.bytesPerPixel), mtl.Region{
			Origin: mtl.Origin{X: 0, Y: 0, Z: 0},
			Size:   mtl.Size{Width: 8, Height: 8, Depth: 1},
		}, 0)
		expected := make([]byte, len(actual))
		for row := range 2 {
			start := ((row+3)*8 + 2) * c.bytesPerPixel
			source := row * 3 * c.bytesPerPixel
			copy(expected[start:start+3*c.bytesPerPixel], pixels[source:source+3*c.bytesPerPixel])
		}
		if !bytes.Equal(actual, expected) {
			t.Fatalf("kind %v pixels = %v, want %v", c.kind, actual, expected)
		}
	}
}

func createAtlas(t *testing.T) *MetalAtlas {
	t.Helper()
	if err := mtl.Load(); err != nil {
		t.Skipf("Metal unavailable: %v", err)
	}
	device, ok := mtl.CreateSystemDefaultDevice()
	if !ok {
		t.Skip("no Metal device")
	}
	defer mtl.Release(device)
	supportsSharedStorage := runtime.GOOS == "ios" || device.SupportsFamily(mtl.GPUFamilyApple1)
	atlas := NewMetalAtlas(device, supportsSharedStorage)
	t.Cleanup(atlas.Release)
	return atlas
}

func makeImageKey(imageID uint, frameIndex int) gpui.AtlasKey {
	return gpui.AtlasKeyFromImage(gpui.RenderImageParams{
		ImageID:    gpui.ImageID(imageID),
		FrameIndex: frameIndex,
	})
}

func insertTile(t *testing.T, atlas *MetalAtlas, key gpui.AtlasKey, size gpui.Size[gpui.DevicePixels]) gpui.AtlasTile {
	t.Helper()
	tile, ok, err := atlas.GetOrInsertWith(key, func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		byteCount := int(size.Width) * int(size.Height) * 4
		return size, make([]byte, byteCount), true, nil
	})
	if err != nil {
		t.Fatalf("inserting atlas tile: %v", err)
	}
	if !ok {
		t.Fatal("retrieving inserted atlas tile")
	}
	return tile
}
