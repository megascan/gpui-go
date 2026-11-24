//go:build linux || js

package wgpu

import (
	"errors"
	"fmt"
	"slices"
	"sync"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/etagere"
	"github.com/megascan/gpui-go/render/wgpu/internal/gpu"
)

func deviceSizeToEtagere(size gpui.Size[gpui.DevicePixels]) etagere.Size {
	return etagere.Size2(int32(size.Width), int32(size.Height))
}

func etagerePointToDevice(point etagere.Point) gpui.Point[gpui.DevicePixels] {
	return gpui.Point[gpui.DevicePixels]{
		X: gpui.DevicePixels(point.X),
		Y: gpui.DevicePixels(point.Y),
	}
}

// WgpuAtlas is the sprite atlas of the wgpu renderer.
type WgpuAtlas struct {
	gpui.PlatformAtlasDefaults
	mu    sync.Mutex
	state *gpui.AtlasState[*wgpuAtlasTextures]
}

type pendingUpload struct {
	id     gpui.AtlasTextureID
	bounds gpui.Bounds[gpui.DevicePixels]
	data   []byte
}

type wgpuAtlasTextures struct {
	device                gpu.Device
	queue                 gpu.Queue
	maxTextureSize        uint32
	colorTextureFormat    gpu.TextureFormat
	storage               wgpuAtlasStorage
	pendingUploads        []pendingUpload
	nextTextureGeneration uint64
}

// WgpuTextureInfo is the view backing an atlas texture.
type WgpuTextureInfo struct {
	View gpu.TextureView
	// Generation distinguishes new textures that reuse an AtlasTextureID.
	Generation uint64
}

// NewWgpuAtlas returns an empty atlas whose textures live on device.
func NewWgpuAtlas(device gpu.Device, queue gpu.Queue, colorTextureFormat gpu.TextureFormat) *WgpuAtlas {
	maxTextureSize := device.Limits().MaxTextureDimension2D
	return &WgpuAtlas{state: gpui.NewAtlasState(&wgpuAtlasTextures{
		device:             device,
		queue:              queue,
		maxTextureSize:     maxTextureSize,
		colorTextureFormat: colorTextureFormat,
	})}
}

// WgpuAtlasFromContext returns an empty atlas on the context's device.
func WgpuAtlasFromContext(context *WgpuContext) *WgpuAtlas {
	return NewWgpuAtlas(context.Device, context.Queue, context.ColorTextureFormat())
}

// BeforeFrame uploads the tiles inserted since the last frame.
func (a *WgpuAtlas) BeforeFrame() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Backend.flushUploads()
}

// GetTextureInfo returns the view backing id, or false once every tile in it has been
// removed. A scene can still reference such a texture when a cached view replays a paint
// from before the image was dropped, so callers must skip those sprites rather than assume
// the texture exists.
func (a *WgpuAtlas) GetTextureInfo(id gpui.AtlasTextureID) (WgpuTextureInfo, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	texture, ok := a.state.Backend.storage.get(id)
	if !ok {
		return WgpuTextureInfo{}, false
	}
	return WgpuTextureInfo{
		View:       texture.view,
		Generation: texture.generation,
	}, true
}

// Clear clears all cached textures and tiles, forcing them to be recreated. Use this for
// incremental recovery when the device is still valid.
func (a *WgpuAtlas) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Clear(func(textures *wgpuAtlasTextures) {
		textures.storage.release()
		textures.storage = wgpuAtlasStorage{}
		textures.pendingUploads = textures.pendingUploads[:0]
	})
}

// HandleDeviceLost clears all textures and cached tiles and moves the atlas to the context's
// new device. The atlas lazily recreates textures as needed on subsequent frames.
func (a *WgpuAtlas) HandleDeviceLost(context *WgpuContext) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Clear(func(textures *wgpuAtlasTextures) {
		textures.device = context.Device
		textures.queue = context.Queue
		textures.colorTextureFormat = context.ColorTextureFormat()
		textures.storage.release()
		textures.storage = wgpuAtlasStorage{}
		textures.pendingUploads = textures.pendingUploads[:0]
	})
}

// GetOrInsertWith returns the key's tile, building and uploading it on a miss.
func (a *WgpuAtlas) GetOrInsertWith(key gpui.AtlasKey, build gpui.AtlasBuildFunc) (gpui.AtlasTile, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state.GetOrInsertWith(key, build)
}

// Remove removes the key's tile, freeing its texture once no tile uses it.
func (a *WgpuAtlas) Remove(key gpui.AtlasKey) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Remove(key)
}

// Insert allocates a tile of kind and queues its upload.
func (t *wgpuAtlasTextures) Insert(kind gpui.AtlasTextureKind, size gpui.Size[gpui.DevicePixels], bytes []byte) (gpui.AtlasTile, error) {
	bytesPerPixel := 4
	if kind == gpui.AtlasTextureKindMonochrome {
		bytesPerPixel = 1
	}
	if size.Width < 0 || size.Height < 0 {
		return gpui.AtlasTile{}, fmt.Errorf("invalid atlas tile size %dx%d", size.Width, size.Height)
	}
	if required := int64(size.Width) * int64(size.Height) * int64(bytesPerPixel); int64(len(bytes)) < required {
		return gpui.AtlasTile{}, fmt.Errorf("atlas tile data too short: %d bytes for %dx%d at %d bytes per pixel, need %d", len(bytes), size.Width, size.Height, bytesPerPixel, required)
	}
	tile, ok := t.allocate(size, kind)
	if !ok {
		return gpui.AtlasTile{}, errors.New("failed to allocate")
	}
	t.uploadTexture(tile.TextureID, tile.Bounds, bytes)
	return tile, nil
}

// Remove frees the tile and drops its texture once it holds no tiles.
func (t *wgpuAtlasTextures) Remove(tile gpui.AtlasTile) {
	id := tile.TextureID
	list := t.storage.index(id.Kind)
	if int(id.Index) >= len(list.Textures) {
		return
	}
	textureSlot := &list.Textures[id.Index]

	if texture, ok := textureSlot.Take().Get(); ok {
		texture.allocator.Deallocate(etagere.AllocIDDeserialize(uint32(tile.TileID)))
		texture.decrementRefCount()
		if texture.isUnreferenced() {
			t.pendingUploads = slices.DeleteFunc(t.pendingUploads, func(upload pendingUpload) bool {
				return upload.id == texture.id
			})
			list.FreeList = append(list.FreeList, int(texture.id.Index))
			texture.release()
		} else {
			*textureSlot = gpuiutil.Some(texture)
		}
	}
}

func (t *wgpuAtlasTextures) allocate(size gpui.Size[gpui.DevicePixels], textureKind gpui.AtlasTextureKind) (gpui.AtlasTile, bool) {
	textures := t.storage.index(textureKind)
	for i := len(textures.Textures) - 1; i >= 0; i-- {
		if texture, ok := textures.Textures[i].Get(); ok {
			if tile, ok := texture.allocate(size); ok {
				return tile, true
			}
		}
	}

	if uint32(size.Width) > t.maxTextureSize || uint32(size.Height) > t.maxTextureSize {
		return gpui.AtlasTile{}, false
	}
	texture := t.pushTexture(size, textureKind)
	tile, ok := texture.allocate(size)
	if !ok {
		t.discardTexture(texture)
	}
	return tile, ok
}

func (t *wgpuAtlasTextures) discardTexture(texture *wgpuAtlasTexture) {
	list := t.storage.index(texture.id.Kind)
	index := int(texture.id.Index)
	if index == len(list.Textures)-1 {
		list.Textures = list.Textures[:index]
	} else {
		list.Textures[index] = gpuiutil.None[*wgpuAtlasTexture]()
		list.FreeList = append(list.FreeList, index)
	}
	texture.release()
}

func (t *wgpuAtlasTextures) pushTexture(minSize gpui.Size[gpui.DevicePixels], kind gpui.AtlasTextureKind) *wgpuAtlasTexture {
	defaultAtlasSize := gpui.Size[gpui.DevicePixels]{
		Width:  1024,
		Height: 1024,
	}
	maxTextureSize := int32(t.maxTextureSize)
	maxAtlasSize := gpui.Size[gpui.DevicePixels]{
		Width:  gpui.DevicePixels(maxTextureSize),
		Height: gpui.DevicePixels(maxTextureSize),
	}

	size := minSize.Min(maxAtlasSize).Max(defaultAtlasSize)
	var format gpu.TextureFormat
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		format = gpu.TextureFormatR8Unorm
	default:
		format = t.colorTextureFormat
	}

	texture := t.device.CreateTexture(&gpu.TextureDescriptor{
		Label: "atlas",
		Size: gpu.Extent3d{
			Width:              uint32(size.Width),
			Height:             uint32(size.Height),
			DepthOrArrayLayers: 1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     gpu.TextureDimensionD2,
		Format:        format,
		Usage:         gpu.TextureUsagesTextureBinding | gpu.TextureUsagesCopyDst,
	})

	view := texture.CreateView(nil)

	textureList := t.storage.index(kind)
	index, hasIndex := 0, false
	if n := len(textureList.FreeList); n > 0 {
		index, hasIndex = textureList.FreeList[n-1], true
		textureList.FreeList = textureList.FreeList[:n-1]
	}
	generation := t.nextTextureGeneration
	t.nextTextureGeneration++

	id := index
	if !hasIndex {
		id = len(textureList.Textures)
	}
	atlasTexture := &wgpuAtlasTexture{
		id: gpui.AtlasTextureID{
			Index: uint32(id),
			Kind:  kind,
		},
		generation: generation,
		allocator:  etagere.NewBucketedAtlasAllocator(deviceSizeToEtagere(size)),
		format:     format,
		texture:    texture,
		view:       view,
	}

	if hasIndex {
		textureList.Textures[index] = gpuiutil.Some(atlasTexture)
	} else {
		textureList.Textures = append(textureList.Textures, gpuiutil.Some(atlasTexture))
	}
	return atlasTexture
}

func (t *wgpuAtlasTextures) uploadTexture(id gpui.AtlasTextureID, bounds gpui.Bounds[gpui.DevicePixels], bytes []byte) {
	var data []byte
	if texture, ok := t.storage.get(id); ok {
		data = swizzleUploadData(bytes, texture.format)
	} else {
		data = slices.Clone(bytes)
	}

	t.pendingUploads = append(t.pendingUploads, pendingUpload{id: id, bounds: bounds, data: data})
}

func (t *wgpuAtlasTextures) flushUploads() {
	uploads := t.pendingUploads
	t.pendingUploads = nil
	for _, upload := range uploads {
		texture, ok := t.storage.get(upload.id)
		if !ok {
			continue
		}
		bytesPerPixel := texture.bytesPerPixel()

		t.queue.WriteTexture(
			gpu.TexelCopyTextureInfo{
				Texture:  texture.texture,
				MipLevel: 0,
				Origin: gpu.Origin3d{
					X: uint32(upload.bounds.Origin.X),
					Y: uint32(upload.bounds.Origin.Y),
					Z: 0,
				},
				Aspect: gpu.TextureAspectAll,
			},
			upload.data,
			gpu.TexelCopyBufferLayout{
				Offset:      0,
				BytesPerRow: uint32(upload.bounds.Size.Width) * uint32(bytesPerPixel),
			},
			gpu.Extent3d{
				Width:              uint32(upload.bounds.Size.Width),
				Height:             uint32(upload.bounds.Size.Height),
				DepthOrArrayLayers: 1,
			},
		)
	}
}

type wgpuAtlasStorage struct {
	monochromeTextures gpui.AtlasTextureList[*wgpuAtlasTexture]
	subpixelTextures   gpui.AtlasTextureList[*wgpuAtlasTexture]
	polychromeTextures gpui.AtlasTextureList[*wgpuAtlasTexture]
}

func (s *wgpuAtlasStorage) index(kind gpui.AtlasTextureKind) *gpui.AtlasTextureList[*wgpuAtlasTexture] {
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		return &s.monochromeTextures
	case gpui.AtlasTextureKindSubpixel:
		return &s.subpixelTextures
	case gpui.AtlasTextureKindPolychrome:
		return &s.polychromeTextures
	}
	panic("gpui: invalid AtlasTextureKind")
}

func (s *wgpuAtlasStorage) get(id gpui.AtlasTextureID) (*wgpuAtlasTexture, bool) {
	list := s.index(id.Kind)
	if int(id.Index) >= len(list.Textures) {
		return nil, false
	}
	return list.Textures[id.Index].Get()
}

func (s *wgpuAtlasStorage) release() {
	for _, list := range []*gpui.AtlasTextureList[*wgpuAtlasTexture]{&s.monochromeTextures, &s.subpixelTextures, &s.polychromeTextures} {
		for texture := range list.IterMut() {
			(*texture).release()
		}
	}
}

type wgpuAtlasTexture struct {
	id            gpui.AtlasTextureID
	generation    uint64
	allocator     *etagere.BucketedAtlasAllocator
	texture       gpu.Texture
	view          gpu.TextureView
	format        gpu.TextureFormat
	liveAtlasKeys uint32
}

func (t *wgpuAtlasTexture) allocate(size gpui.Size[gpui.DevicePixels]) (gpui.AtlasTile, bool) {
	allocation, ok := t.allocator.Allocate(deviceSizeToEtagere(size))
	if !ok {
		return gpui.AtlasTile{}, false
	}
	tile := gpui.AtlasTile{
		TextureID: t.id,
		TileID:    gpui.TileID(allocation.ID.Serialize()),
		Padding:   0,
		Bounds: gpui.Bounds[gpui.DevicePixels]{
			Origin: etagerePointToDevice(allocation.Rectangle.Min),
			Size:   size,
		},
	}
	t.liveAtlasKeys++
	return tile, true
}

func (t *wgpuAtlasTexture) bytesPerPixel() uint8 {
	switch t.format {
	case gpu.TextureFormatR8Unorm:
		return 1
	case gpu.TextureFormatBgra8Unorm, gpu.TextureFormatRgba8Unorm:
		return 4
	}
	return 4
}

func (t *wgpuAtlasTexture) decrementRefCount() {
	t.liveAtlasKeys--
}

func (t *wgpuAtlasTexture) isUnreferenced() bool {
	return t.liveAtlasKeys == 0
}

func (t *wgpuAtlasTexture) release() {
	t.view.Release()
	t.texture.Release()
}

func swizzleUploadData(bytes []byte, format gpu.TextureFormat) []byte {
	switch format {
	case gpu.TextureFormatRgba8Unorm:
		data := slices.Clone(bytes)
		for i := 0; i+4 <= len(data); i += 4 {
			data[i], data[i+2] = data[i+2], data[i]
		}
		return data
	}
	return slices.Clone(bytes)
}
