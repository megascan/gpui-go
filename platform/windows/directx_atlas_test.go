//go:build windows

package windows

import (
	"testing"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/internal/win/d3d"
)

func createAtlas() (*DirectXAtlas, bool) {
	device, _, deviceContext, err := d3d.D3D11CreateDevice(nil, d3d.D3D_DRIVER_TYPE_WARP, 0, d3d.D3D11_CREATE_DEVICE_BGRA_SUPPORT, nil, d3d.D3D11_SDK_VERSION)
	if err != nil {
		return nil, false
	}
	defer device.Release()
	defer deviceContext.Release()
	return NewDirectXAtlas(device, deviceContext), true
}

func makeImageKey(imageID uint) gpui.AtlasKey {
	return gpui.AtlasKeyFromImage(gpui.RenderImageParams{
		ImageID:    gpui.ImageID(imageID),
		FrameIndex: 0,
	})
}

func insertTile(t *testing.T, atlas *DirectXAtlas, key gpui.AtlasKey, size gpui.Size[gpui.DevicePixels]) gpui.AtlasTile {
	t.Helper()
	tile, ok, err := atlas.GetOrInsertWith(key, func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		byteCount := int(size.Width) * int(size.Height) * 4
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

func TestDirectxAtlas_RemoveDeallocatesTileSpaceForReuse(t *testing.T) {
	atlas, ok := createAtlas()
	if !ok {
		return
	}
	defer atlas.Release()

	small := gpui.Size[gpui.DevicePixels]{Width: 64, Height: 64}
	big := gpui.Size[gpui.DevicePixels]{Width: 700, Height: 700}

	keeperKey := makeImageKey(1)
	bigKeyA := makeImageKey(2)
	bigKeyB := makeImageKey(3)

	keeperTile := insertTile(t, atlas, keeperKey, small)
	tileA := insertTile(t, atlas, bigKeyA, big)
	if keeperTile.TextureID != tileA.TextureID {
		t.Fatalf("assertion `left == right` failed\n  left: %+v\n right: %+v", keeperTile.TextureID, tileA.TextureID)
	}

	atlas.Remove(bigKeyA)

	tileB := insertTile(t, atlas, bigKeyB, big)
	if tileB.TextureID != keeperTile.TextureID {
		t.Fatalf("assertion `left == right` failed\n  left: %+v\n right: %+v", tileB.TextureID, keeperTile.TextureID)
	}
}
