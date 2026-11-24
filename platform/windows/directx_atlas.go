//go:build windows

package windows

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/etagere"
	"github.com/megascan/gpui-go/internal/win/d3d"
)

// DirectXAtlas is the sprite atlas of the DirectX renderer.
type DirectXAtlas struct {
	gpui.PlatformAtlasDefaults
	mu    sync.Mutex
	state *gpui.AtlasState[*directXAtlasTextures]
}

type directXAtlasTextures struct {
	device             *d3d.ID3D11Device
	deviceContext      *d3d.ID3D11DeviceContext
	monochromeTextures gpui.AtlasTextureList[*directXAtlasTexture]
	polychromeTextures gpui.AtlasTextureList[*directXAtlasTexture]
	subpixelTextures   gpui.AtlasTextureList[*directXAtlasTexture]
}

type directXAtlasTexture struct {
	id            gpui.AtlasTextureID
	bytesPerPixel uint32
	allocator     *etagere.BucketedAtlasAllocator
	texture       *d3d.ID3D11Texture2D
	view          [1]*d3d.ID3D11ShaderResourceView
	liveAtlasKeys uint32
}

// NewDirectXAtlas returns an empty atlas whose textures device allocates; it adds a reference to
// device and deviceContext.
func NewDirectXAtlas(device *d3d.ID3D11Device, deviceContext *d3d.ID3D11DeviceContext) *DirectXAtlas {
	return &DirectXAtlas{state: gpui.NewAtlasState(&directXAtlasTextures{
		device:        device.Clone(),
		deviceContext: deviceContext.Clone(),
	})}
}

// GetTextureView returns the view backing id, AddRef'd, or false once every tile in it has been
// removed. A scene can still reference such a texture when a cached view replays a paint from
// before the image was dropped, so callers must skip those sprites rather than assume the
// texture exists.
func (a *DirectXAtlas) GetTextureView(id gpui.AtlasTextureID) ([1]*d3d.ID3D11ShaderResourceView, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	texture, ok := a.state.Backend.texture(id)
	if !ok {
		return [1]*d3d.ID3D11ShaderResourceView{}, false
	}
	view := texture.view
	if view[0] != nil {
		view[0].AddRef()
	}
	return view, true
}

// HandleDeviceLost forgets every tile and texture and switches to the new device.
func (a *DirectXAtlas) HandleDeviceLost(device *d3d.ID3D11Device, deviceContext *d3d.ID3D11DeviceContext) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Clear(func(textures *directXAtlasTextures) {
		newDevice := device.Clone()
		newContext := deviceContext.Clone()
		textures.releaseAll()
		textures.device = newDevice
		textures.deviceContext = newContext
	})
}

// Release drops every texture, the device and the context, as dropping the Rust atlas does. It is
// idempotent.
func (a *DirectXAtlas) Release() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Backend.releaseAll()
}

// GetOrInsertWith returns the key's tile, building and uploading it on a miss.
func (a *DirectXAtlas) GetOrInsertWith(key gpui.AtlasKey, build gpui.AtlasBuildFunc) (gpui.AtlasTile, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state.GetOrInsertWith(key, build)
}

// Remove removes the key's tile, freeing its texture once no tile uses it.
func (a *DirectXAtlas) Remove(key gpui.AtlasKey) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Remove(key)
}

func (t *directXAtlasTextures) releaseAll() {
	for _, list := range []*gpui.AtlasTextureList[*directXAtlasTexture]{&t.monochromeTextures, &t.polychromeTextures, &t.subpixelTextures} {
		for _, slot := range list.Drain() {
			if texture, ok := slot.Get(); ok {
				texture.release()
			}
		}
		*list = gpui.AtlasTextureList[*directXAtlasTexture]{}
	}
	if t.deviceContext != nil {
		t.deviceContext.Release()
		t.deviceContext = nil
	}
	if t.device != nil {
		t.device.Release()
		t.device = nil
	}
}

func (t *directXAtlasTextures) list(kind gpui.AtlasTextureKind) *gpui.AtlasTextureList[*directXAtlasTexture] {
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		return &t.monochromeTextures
	case gpui.AtlasTextureKindPolychrome:
		return &t.polychromeTextures
	case gpui.AtlasTextureKindSubpixel:
		return &t.subpixelTextures
	}
	panic("internal error: entered unreachable code")
}

// Insert allocates a tile of kind and uploads bytes into it.
func (t *directXAtlasTextures) Insert(kind gpui.AtlasTextureKind, size gpui.Size[gpui.DevicePixels], bytes []byte) (gpui.AtlasTile, error) {
	tile, ok := t.allocate(size, kind)
	if !ok {
		return gpui.AtlasTile{}, errors.New("failed to allocate")
	}
	texture, ok := t.texture(tile.TextureID)
	if !ok {
		return gpui.AtlasTile{}, errors.New("allocated tile refers to a missing texture")
	}
	texture.upload(t.deviceContext, tile.Bounds, bytes)
	return tile, nil
}

// Remove frees the tile and drops its texture once it holds no tiles.
func (t *directXAtlasTextures) Remove(tile gpui.AtlasTile) {
	id := tile.TextureID
	textures := t.list(id.Kind)
	if int(id.Index) >= len(textures.Textures) {
		return
	}
	textureSlot := &textures.Textures[id.Index]
	if texture, ok := textureSlot.Take().Get(); ok {
		texture.allocator.Deallocate(etagere.AllocIDDeserialize(uint32(tile.TileID)))
		texture.decrementRefCount()
		if texture.isUnreferenced() {
			textures.FreeList = append(textures.FreeList, int(texture.id.Index))
			texture.release()
		} else {
			*textureSlot = gpuiutil.Some(texture)
		}
	}
}

func (t *directXAtlasTextures) allocate(size gpui.Size[gpui.DevicePixels], textureKind gpui.AtlasTextureKind) (gpui.AtlasTile, bool) {
	textures := t.list(textureKind)
	for i := len(textures.Textures) - 1; i >= 0; i-- {
		if texture, ok := textures.Textures[i].Get(); ok {
			if tile, ok := texture.allocate(size); ok {
				return tile, true
			}
		}
	}
	texture, ok := t.pushTexture(size, textureKind)
	if !ok {
		return gpui.AtlasTile{}, false
	}
	return texture.allocate(size)
}

func (t *directXAtlasTextures) pushTexture(minSize gpui.Size[gpui.DevicePixels], kind gpui.AtlasTextureKind) (*directXAtlasTexture, bool) {
	defaultAtlasSize := gpui.Size[gpui.DevicePixels]{Width: 1024, Height: 1024}
	maxAtlasSize := gpui.Size[gpui.DevicePixels]{Width: 16384, Height: 16384}
	size := minSize.Min(maxAtlasSize).Max(defaultAtlasSize)
	var pixelFormat, bindFlag, bytesPerPixel uint32
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		pixelFormat = d3d.DXGI_FORMAT_R8_UNORM
		bindFlag = d3d.D3D11_BIND_SHADER_RESOURCE
		bytesPerPixel = 1
	case gpui.AtlasTextureKindPolychrome:
		pixelFormat = d3d.DXGI_FORMAT_B8G8R8A8_UNORM
		bindFlag = d3d.D3D11_BIND_SHADER_RESOURCE
		bytesPerPixel = 4
	case gpui.AtlasTextureKindSubpixel:
		pixelFormat = d3d.DXGI_FORMAT_R8G8B8A8_UNORM
		bindFlag = d3d.D3D11_BIND_SHADER_RESOURCE
		bytesPerPixel = 4
	default:
		panic("internal error: entered unreachable code")
	}
	textureDesc := d3d.D3D11_TEXTURE2D_DESC{
		Width:     uint32(size.Width),
		Height:    uint32(size.Height),
		MipLevels: 1,
		ArraySize: 1,
		Format:    pixelFormat,
		SampleDesc: d3d.DXGI_SAMPLE_DESC{
			Count:   1,
			Quality: 0,
		},
		Usage:          d3d.D3D11_USAGE_DEFAULT,
		BindFlags:      bindFlag,
		CPUAccessFlags: 0,
		MiscFlags:      0,
	}
	texture, err := t.device.CreateTexture2D(&textureDesc, nil)
	if err != nil {
		return nil, false
	}

	textureList := t.list(kind)
	index := -1
	if n := len(textureList.FreeList); n > 0 {
		index = textureList.FreeList[n-1]
		textureList.FreeList = textureList.FreeList[:n-1]
	}
	view, err := t.device.CreateShaderResourceView(texture.Resource())
	if err != nil {
		texture.Release()
		return nil, false
	}
	atlasIndex := index
	if atlasIndex < 0 {
		atlasIndex = len(textureList.Textures)
	}
	atlasTexture := &directXAtlasTexture{
		id: gpui.AtlasTextureID{
			Index: uint32(atlasIndex),
			Kind:  kind,
		},
		bytesPerPixel: bytesPerPixel,
		allocator:     etagere.NewBucketedAtlasAllocator(deviceSizeToEtagere(size)),
		texture:       texture,
		view:          [1]*d3d.ID3D11ShaderResourceView{view},
		liveAtlasKeys: 0,
	}
	if index >= 0 {
		textureList.Textures[index] = gpuiutil.Some(atlasTexture)
	} else {
		textureList.Textures = append(textureList.Textures, gpuiutil.Some(atlasTexture))
	}
	return atlasTexture, true
}

func (t *directXAtlasTextures) texture(id gpui.AtlasTextureID) (*directXAtlasTexture, bool) {
	textures := t.list(id.Kind)
	if int(id.Index) >= len(textures.Textures) {
		return nil, false
	}
	return textures.Textures[id.Index].Get()
}

func (t *directXAtlasTexture) release() {
	if t.view[0] != nil {
		t.view[0].Release()
		t.view[0] = nil
	}
	if t.texture != nil {
		t.texture.Release()
		t.texture = nil
	}
}

func (t *directXAtlasTexture) allocate(size gpui.Size[gpui.DevicePixels]) (gpui.AtlasTile, bool) {
	allocation, ok := t.allocator.Allocate(deviceSizeToEtagere(size))
	if !ok {
		return gpui.AtlasTile{}, false
	}
	tile := gpui.AtlasTile{
		TextureID: t.id,
		TileID:    gpui.TileID(allocation.ID.Serialize()),
		Bounds: gpui.Bounds[gpui.DevicePixels]{
			Origin: etagerePointToDevice(allocation.Rectangle.Min),
			Size:   size,
		},
		Padding: 0,
	}
	t.liveAtlasKeys++
	return tile, true
}

func (t *directXAtlasTexture) upload(deviceContext *d3d.ID3D11DeviceContext, bounds gpui.Bounds[gpui.DevicePixels], bytes []byte) {
	rowBytes := uint64(bounds.Size.Width.ToBytes(uint8(t.bytesPerPixel)))
	expected := rowBytes * uint64(max(bounds.Size.Height, 0))
	if uint64(len(bytes)) < expected {
		slog.Error(fmt.Sprintf(
			"DirectXAtlasTexture::upload: source slice is %d bytes but the %dx%d region requires %d bytes; skipping upload to avoid a driver over-read",
			len(bytes),
			int32(bounds.Size.Width),
			int32(bounds.Size.Height),
			expected,
		))
		return
	}
	var data unsafe.Pointer
	if len(bytes) > 0 {
		data = unsafe.Pointer(&bytes[0])
	}
	deviceContext.UpdateSubresource(
		t.texture.Resource(),
		0,
		&d3d.D3D11_BOX{
			Left:   uint32(bounds.Left()),
			Top:    uint32(bounds.Top()),
			Front:  0,
			Right:  uint32(bounds.Right()),
			Bottom: uint32(bounds.Bottom()),
			Back:   1,
		},
		data,
		bounds.Size.Width.ToBytes(uint8(t.bytesPerPixel)),
		0,
	)
}

func (t *directXAtlasTexture) decrementRefCount() {
	t.liveAtlasKeys--
}

func (t *directXAtlasTexture) isUnreferenced() bool {
	return t.liveAtlasKeys == 0
}

func deviceSizeToEtagere(size gpui.Size[gpui.DevicePixels]) etagere.Size {
	return etagere.Size2(int32(size.Width), int32(size.Height))
}

func etagerePointToDevice(value etagere.Point) gpui.Point[gpui.DevicePixels] {
	return gpui.Point[gpui.DevicePixels]{
		X: gpui.DevicePixels(value.X),
		Y: gpui.DevicePixels(value.Y),
	}
}
