//go:build darwin

package apple

import (
	"errors"
	"fmt"
	"math"
	"math/bits"
	"sync"
	"unsafe"

	gpui "github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/internal/apple/mtl"
	"github.com/megascan/gpui-go/internal/etagere"
)

// MetalAtlas is the sprite atlas of the Metal renderer.
type MetalAtlas struct {
	gpui.PlatformAtlasDefaults
	mu    sync.Mutex
	state *gpui.AtlasState[*metalAtlasTextures]
}

// NewMetalAtlas returns an empty atlas whose textures device allocates; it retains device.
func NewMetalAtlas(device mtl.Device, supportsSharedStorage bool) *MetalAtlas {
	return &MetalAtlas{state: gpui.NewAtlasState(&metalAtlasTextures{
		device:                mtl.Retain(device),
		supportsSharedStorage: supportsSharedStorage,
	})}
}

// MetalTexture returns a +1 reference to the GPU texture backing id, or false once every tile in
// it has been removed. A scene can still reference such a texture when a cached view replays a
// paint from before the image was dropped, so callers must skip those sprites rather than assume
// the texture exists.
func (a *MetalAtlas) MetalTexture(id gpui.AtlasTextureID) (mtl.Texture, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	texture, ok := a.state.Backend.texture(id)
	if !ok {
		return 0, false
	}
	return mtl.Retain(texture.metalTexture), true
}

// GetOrInsertWith returns the key's tile, building and uploading it on a miss.
func (a *MetalAtlas) GetOrInsertWith(key gpui.AtlasKey, build gpui.AtlasBuildFunc) (gpui.AtlasTile, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.state.GetOrInsertWith(key, build)
}

// Remove removes the key's tile, freeing its texture once no tile uses it.
func (a *MetalAtlas) Remove(key gpui.AtlasKey) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.state.Remove(key)
}

// Release drops every texture and the device, as dropping the Rust atlas does. It is idempotent.
func (a *MetalAtlas) Release() {
	a.mu.Lock()
	defer a.mu.Unlock()
	t := a.state.Backend
	if t.device == 0 {
		return
	}
	for _, list := range []*gpui.AtlasTextureList[*metalAtlasTexture]{&t.monochromeTextures, &t.polychromeTextures} {
		for _, slot := range list.Drain() {
			if texture, ok := slot.Get(); ok {
				texture.release()
			}
		}
	}
	mtl.Release(t.device)
	t.device = 0
}

type metalAtlasTextures struct {
	device                mtl.Device
	supportsSharedStorage bool
	monochromeTextures    gpui.AtlasTextureList[*metalAtlasTexture]
	polychromeTextures    gpui.AtlasTextureList[*metalAtlasTexture]
}

func (t *metalAtlasTextures) list(kind gpui.AtlasTextureKind) *gpui.AtlasTextureList[*metalAtlasTexture] {
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		return &t.monochromeTextures
	case gpui.AtlasTextureKindPolychrome:
		return &t.polychromeTextures
	}
	panic("internal error: entered unreachable code")
}

// Insert validates the upload, allocates a tile of kind and copies bytes into it.
func (t *metalAtlasTextures) Insert(kind gpui.AtlasTextureKind, size gpui.Size[gpui.DevicePixels], bytes []byte) (gpui.AtlasTile, error) {
	upload, err := newAtlasUpload(kind, size, bytes)
	if err != nil {
		return gpui.AtlasTile{}, err
	}
	tile, err := t.allocate(size, kind)
	if err != nil {
		return gpui.AtlasTile{}, err
	}
	texture, ok := t.texture(tile.TextureID)
	if !ok {
		return gpui.AtlasTile{}, errors.New("looking up allocated atlas texture")
	}
	if err := texture.upload(tile.Bounds.Origin, &upload); err != nil {
		return gpui.AtlasTile{}, err
	}
	return tile, nil
}

// Remove frees the tile and drops its texture once it holds no tiles.
func (t *metalAtlasTextures) Remove(tile gpui.AtlasTile) {
	id := tile.TextureID
	textures := t.list(id.Kind)
	var textureSlot *gpui.Option[*metalAtlasTexture]
	for i := range textures.Textures {
		if texture, ok := textures.Textures[i].Get(); ok && texture.id == id {
			textureSlot = &textures.Textures[i]
			break
		}
	}
	if textureSlot == nil {
		return
	}
	if texture, ok := textureSlot.Take().Get(); ok {
		texture.allocator.Deallocate(etagere.AllocIDDeserialize(uint32(tile.TileID)))
		texture.decrementRefCount()
		if texture.isUnreferenced() {
			textures.FreeList = append(textures.FreeList, int(id.Index))
			texture.release()
		} else {
			*textureSlot = gpuiutil.Some(texture)
		}
	}
}

func (t *metalAtlasTextures) allocate(size gpui.Size[gpui.DevicePixels], textureKind gpui.AtlasTextureKind) (gpui.AtlasTile, error) {
	textures := t.list(textureKind)
	for i := len(textures.Textures) - 1; i >= 0; i-- {
		if texture, ok := textures.Textures[i].Get(); ok {
			if tile, ok := texture.allocate(size); ok {
				return tile, nil
			}
		}
	}
	texture, err := t.pushTexture(size, textureKind)
	if err != nil {
		return gpui.AtlasTile{}, err
	}
	tile, ok := texture.allocate(size)
	if !ok {
		return gpui.AtlasTile{}, errors.New("allocating atlas tile")
	}
	return tile, nil
}

func (t *metalAtlasTextures) pushTexture(minSize gpui.Size[gpui.DevicePixels], kind gpui.AtlasTextureKind) (*metalAtlasTexture, error) {
	defaultAtlasSize := gpui.Size[gpui.DevicePixels]{Width: 1024, Height: 1024}
	maxAtlasSize := gpui.Size[gpui.DevicePixels]{Width: 16384, Height: 16384}
	size := minSize.Min(maxAtlasSize).Max(defaultAtlasSize)
	textureDescriptor := mtl.NewTextureDescriptor()
	defer mtl.Release(textureDescriptor)
	textureDescriptor.SetWidth(uint(size.Width))
	textureDescriptor.SetHeight(uint(size.Height))
	var pixelFormat mtl.PixelFormat
	var usage mtl.TextureUsage
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		pixelFormat = mtl.PixelFormatA8Unorm
		usage = mtl.TextureUsageShaderRead
	case gpui.AtlasTextureKindPolychrome:
		pixelFormat = mtl.PixelFormatBGRA8Unorm
		usage = mtl.TextureUsageShaderRead
	default:
		panic("internal error: entered unreachable code")
	}
	textureDescriptor.SetPixelFormat(pixelFormat)
	textureDescriptor.SetUsage(usage)
	if t.supportsSharedStorage {
		textureDescriptor.SetStorageMode(mtl.StorageModeShared)
	} else {
		textureDescriptor.SetStorageMode(mtl.StorageModeManaged)
	}
	metalTexture := t.device.NewTextureWithDescriptor(textureDescriptor)
	if metalTexture == 0 {
		return nil, errors.New("allocating atlas texture")
	}

	textureList := t.list(kind)
	var index int
	if n := len(textureList.FreeList); n > 0 {
		index = textureList.FreeList[n-1]
		textureList.FreeList = textureList.FreeList[:n-1]
	} else {
		index = len(textureList.Textures)
		textureList.Textures = append(textureList.Textures, gpuiutil.None[*metalAtlasTexture]())
	}

	atlasTexture := &metalAtlasTexture{
		id: gpui.AtlasTextureID{
			Index: uint32(index),
			Kind:  kind,
		},
		allocator:    etagere.NewBucketedAtlasAllocator(sizeToEtagere(size)),
		metalTexture: metalTexture,
	}
	if old, ok := textureList.Textures[index].Get(); ok {
		old.release()
	}
	textureList.Textures[index] = gpuiutil.Some(atlasTexture)
	return atlasTexture, nil
}

func (t *metalAtlasTextures) texture(id gpui.AtlasTextureID) (*metalAtlasTexture, bool) {
	textures := t.list(id.Kind)
	if int(id.Index) >= len(textures.Textures) {
		return nil, false
	}
	return textures.Textures[id.Index].Get()
}

type metalAtlasTexture struct {
	id            gpui.AtlasTextureID
	allocator     *etagere.BucketedAtlasAllocator
	metalTexture  mtl.Texture
	liveAtlasKeys uint32
	released      sync.Once
}

func (t *metalAtlasTexture) release() {
	t.released.Do(func() { mtl.Release(t.metalTexture) })
}

func (t *metalAtlasTexture) allocate(size gpui.Size[gpui.DevicePixels]) (gpui.AtlasTile, bool) {
	allocation, ok := t.allocator.Allocate(sizeToEtagere(size))
	if !ok {
		return gpui.AtlasTile{}, false
	}
	tile := gpui.AtlasTile{
		TextureID: t.id,
		TileID:    gpui.TileID(allocation.ID.Serialize()),
		Bounds: gpui.Bounds[gpui.DevicePixels]{
			Origin: pointFromEtagere(allocation.Rectangle.Min),
			Size:   size,
		},
		Padding: 0,
	}
	t.liveAtlasKeys++
	return tile, true
}

func (t *metalAtlasTexture) upload(origin gpui.Point[gpui.DevicePixels], upload *atlasUpload) error {
	if t.id.Kind != upload.kind {
		return errors.New("atlas upload kind does not match texture")
	}
	if !(origin.X >= 0 && origin.Y >= 0 &&
		checkedAddLE(uint64(origin.X), uint64(upload.size.Width), uint64(t.metalTexture.Width())) &&
		checkedAddLE(uint64(origin.Y), uint64(upload.size.Height), uint64(t.metalTexture.Height()))) {
		return errors.New("atlas upload is outside texture bounds")
	}
	region := mtl.Region{
		Origin: mtl.Origin{X: uint(origin.X), Y: uint(origin.Y), Z: 0},
		Size:   mtl.Size{Width: uint(upload.size.Width), Height: uint(upload.size.Height), Depth: 1},
	}
	var bytes unsafe.Pointer
	if len(upload.bytes) > 0 {
		bytes = unsafe.Pointer(&upload.bytes[0])
	}
	t.metalTexture.ReplaceRegion(region, 0, bytes, upload.bytesPerRow)
	return nil
}

func checkedAddLE(a, b, limit uint64) bool {
	end, carry := bits.Add64(a, b, 0)
	return carry == 0 && end <= limit
}

func (t *metalAtlasTexture) decrementRefCount() {
	t.liveAtlasKeys--
}

func (t *metalAtlasTexture) isUnreferenced() bool {
	return t.liveAtlasKeys == 0
}

type atlasUpload struct {
	kind        gpui.AtlasTextureKind
	size        gpui.Size[gpui.DevicePixels]
	bytes       []byte
	bytesPerRow uint
}

func newAtlasUpload(kind gpui.AtlasTextureKind, size gpui.Size[gpui.DevicePixels], bytes []byte) (atlasUpload, error) {
	if !(size.Width > 0 && size.Height > 0) {
		return atlasUpload{}, errors.New("atlas dimensions must be positive")
	}
	var bytesPerPixel uint64
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		bytesPerPixel = 1
	case gpui.AtlasTextureKindPolychrome:
		bytesPerPixel = 4
	default:
		panic("internal error: entered unreachable code")
	}
	hi, bytesPerRow := bits.Mul64(uint64(size.Width), bytesPerPixel)
	if hi != 0 || bytesPerRow > math.MaxUint {
		return atlasUpload{}, errors.New("calculating atlas row byte count")
	}
	hi, byteCount := bits.Mul64(bytesPerRow, uint64(size.Height))
	if hi != 0 || byteCount > math.MaxUint {
		return atlasUpload{}, errors.New("calculating atlas byte count")
	}
	if uint64(len(bytes)) < byteCount {
		return atlasUpload{}, fmt.Errorf("atlas upload byte length is %d, expected at least %d", len(bytes), byteCount)
	}
	return atlasUpload{
		kind:        kind,
		size:        size,
		bytes:       bytes,
		bytesPerRow: uint(bytesPerRow),
	}, nil
}

func sizeToEtagere(size gpui.Size[gpui.DevicePixels]) etagere.Size {
	return etagere.Size2(int32(size.Width), int32(size.Height))
}

func pointFromEtagere(value etagere.Point) gpui.Point[gpui.DevicePixels] {
	return gpui.Point[gpui.DevicePixels]{
		X: gpui.DevicePixels(value.X),
		Y: gpui.DevicePixels(value.Y),
	}
}
