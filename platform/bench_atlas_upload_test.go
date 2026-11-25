package platform_test

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/gpuiutil"
	"github.com/megascan/gpui-go/platform"
)

type atlasUploadEntry struct {
	key   gpui.AtlasKey
	bytes []byte
}

type atlasUploadFixture struct {
	renderer   gpui.PlatformHeadlessRenderer
	atlas      gpui.PlatformAtlas
	dimensions gpui.Size[gpui.DevicePixels]
	seed       atlasUploadEntry
	entries    []atlasUploadEntry
}

type atlasUploadBatch struct {
	fixture      *atlasUploadFixture
	textureID    gpui.AtlasTextureID
	misses       bool
	builderCalls int
	tiles        []gpui.AtlasTile
}

func BenchmarkAtlasUpload_AtlasUpload(b *testing.B) {
	if runtime.GOOS != "darwin" {
		b.Skip("this benchmark requires Metal")
	}
	if gpuiutil.DebugAssertions {
		b.Skip("use -tags gpui_release")
	}
	if err := headlessRendererProbe(); err != nil {
		b.Skipf("no GPU adapter: %v", err)
	}
	for _, format := range []struct {
		kind gpui.AtlasTextureKind
		name string
	}{
		{gpui.AtlasTextureKindMonochrome, "a8"},
		{gpui.AtlasTextureKindPolychrome, "bgra8"},
	} {
		for _, shape := range []struct{ width, height, count int }{{1, 1, 128}, {3, 2, 128}, {16, 16, 128}, {256, 256, 8}} {
			dimensions := gpui.NewSize(gpui.DevicePixels(shape.width), gpui.DevicePixels(shape.height))
			fixture, err := newAtlasUploadFixture(format.kind, dimensions, shape.count)
			if err != nil {
				panic(fmt.Sprintf("create Metal fixture: %v", err))
			}
			if err := fixture.validatePixels(); err != nil {
				panic(fmt.Sprintf("validate atlas pixels: %v", err))
			}
			for _, mode := range []struct {
				misses bool
				name   string
			}{{true, "miss"}, {false, "hit"}} {
				id := fmt.Sprintf("atlas_upload/%s_%s/%dx%d_%d", mode.name, format.name, shape.width, shape.height, shape.count)
				b.Run(id, func(b *testing.B) {
					for b.Loop() {
						b.StopTimer()
						batch, err := fixture.prepare(mode.misses)
						if err != nil {
							panic(fmt.Sprintf("prepare atlas batch: %v", err))
						}
						b.StartTimer()
						if err := batch.run(); err != nil {
							panic(fmt.Sprintf("insert atlas batch: %v", err))
						}
						b.StopTimer()
						batch.drop()
						b.StartTimer()
					}
					b.ReportMetric(float64(shape.count), "elements/op")
				})
			}
		}
	}
}

func newAtlasUploadFixture(kind gpui.AtlasTextureKind, dimensions gpui.Size[gpui.DevicePixels], count int) (*atlasUploadFixture, error) {
	renderer, err := platform.CurrentHeadlessRenderer()
	if err != nil {
		return nil, err
	}
	if renderer == nil {
		return nil, errors.New("Metal unavailable")
	}
	fixture := &atlasUploadFixture{
		renderer:   renderer,
		atlas:      renderer.SpriteAtlas(),
		dimensions: dimensions,
		seed:       makeAtlasUploadEntry(kind, dimensions, count),
	}
	for index := range count {
		fixture.entries = append(fixture.entries, makeAtlasUploadEntry(kind, dimensions, index))
	}
	return fixture, nil
}

func (f *atlasUploadFixture) insert(entry *atlasUploadEntry, builderCalls *int) (gpui.AtlasTile, error) {
	tile, ok, err := f.atlas.GetOrInsertWith(entry.key, func() (gpui.Size[gpui.DevicePixels], []byte, bool, error) {
		*builderCalls++
		return f.dimensions, entry.bytes, true, nil
	})
	if err != nil {
		return gpui.AtlasTile{}, err
	}
	if !ok {
		return gpui.AtlasTile{}, errors.New("missing atlas tile")
	}
	return tile, nil
}

func (f *atlasUploadFixture) prepare(misses bool) (*atlasUploadBatch, error) {
	for i := range f.entries {
		f.atlas.Remove(f.entries[i].key)
	}
	f.atlas.Remove(f.seed.key)
	builderCalls := 0
	seed, err := f.insert(&f.seed, &builderCalls)
	if err != nil {
		return nil, err
	}
	if builderCalls != 1 {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: 1", builderCalls))
	}
	if !misses {
		for i := range f.entries {
			tile, err := f.insert(&f.entries[i], &builderCalls)
			if err != nil {
				return nil, err
			}
			if tile.TextureID != seed.TextureID {
				panic(fmt.Sprintf("assertion `left == right` failed\n  left: %v\n right: %v", tile.TextureID, seed.TextureID))
			}
		}
		if builderCalls != len(f.entries)+1 {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", builderCalls, len(f.entries)+1))
		}
	}
	return &atlasUploadBatch{
		fixture:   f,
		textureID: seed.TextureID,
		misses:    misses,
		tiles:     make([]gpui.AtlasTile, 0, len(f.entries)),
	}, nil
}

func (f *atlasUploadFixture) validatePixels() error {
	batch, err := f.prepare(true)
	if err != nil {
		return err
	}
	if err := batch.run(); err != nil {
		return err
	}
	tiles := slices.Clone(batch.tiles)
	batch.drop()
	for i, tile := range tiles {
		entry := &f.entries[i]
		scene := atlasUploadTileScene(tile)
		image, err := f.renderer.RenderSceneToImage(scene, f.dimensions)
		if err != nil {
			return err
		}
		var expected []byte
		switch tile.TextureID.Kind {
		case gpui.AtlasTextureKindMonochrome:
			for _, alpha := range entry.bytes {
				expected = append(expected, alpha, alpha, alpha, 255)
			}
		case gpui.AtlasTextureKindPolychrome:
			for pixel := range slices.Chunk(entry.bytes, 4) {
				expected = append(expected, pixel[2], pixel[1], pixel[0], 255)
			}
		default:
			panic("internal error: entered unreachable code")
		}
		if !slices.Equal(image.Pix, expected) {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %v\n right: %v", image.Pix, expected))
		}
	}
	batch, err = f.prepare(false)
	if err != nil {
		return err
	}
	if err := batch.run(); err != nil {
		return err
	}
	batch.drop()
	return nil
}

func (bt *atlasUploadBatch) run() error {
	for i := range bt.fixture.entries {
		tile, err := bt.fixture.insert(&bt.fixture.entries[i], &bt.builderCalls)
		if err != nil {
			return err
		}
		bt.tiles = append(bt.tiles, tile)
	}
	runtime.KeepAlive(bt.tiles)
	return nil
}

func (bt *atlasUploadBatch) drop() {
	count := len(bt.fixture.entries)
	if len(bt.tiles) != count {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", len(bt.tiles), count))
	}
	expectedCalls := 0
	if bt.misses {
		expectedCalls = count
	}
	if bt.builderCalls != expectedCalls {
		panic(fmt.Sprintf("assertion `left == right` failed\n  left: %d\n right: %d", bt.builderCalls, expectedCalls))
	}
	for _, tile := range bt.tiles {
		if tile.TextureID != bt.textureID {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %v\n right: %v", tile.TextureID, bt.textureID))
		}
		if tile.Bounds.Size != bt.fixture.dimensions {
			panic(fmt.Sprintf("assertion `left == right` failed\n  left: %v\n right: %v", tile.Bounds.Size, bt.fixture.dimensions))
		}
	}
}

func makeAtlasUploadEntry(kind gpui.AtlasTextureKind, dimensions gpui.Size[gpui.DevicePixels], index int) atlasUploadEntry {
	var key gpui.AtlasKey
	switch kind {
	case gpui.AtlasTextureKindMonochrome:
		key = gpui.AtlasKeyFromSvg(gpui.RenderSvgParams{
			Path: gpui.SharedString(fmt.Sprintf("atlas-upload-%d", index)),
			Size: dimensions,
		})
	case gpui.AtlasTextureKindPolychrome:
		key = gpui.AtlasKeyFromImage(gpui.RenderImageParams{ImageID: gpui.ImageID(index), FrameIndex: 0})
	default:
		panic("internal error: entered unreachable code")
	}
	var bytes []byte
	for row := range int(dimensions.Height) {
		for column := range int(dimensions.Width) {
			value := uint8(index)*73 + uint8(row)*31 + uint8(column)*17
			switch kind {
			case gpui.AtlasTextureKindMonochrome:
				bytes = append(bytes, value)
			case gpui.AtlasTextureKindPolychrome:
				bytes = append(bytes, value, value+53, value+107, 255)
			default:
				panic("internal error: entered unreachable code")
			}
		}
	}
	return atlasUploadEntry{key: key, bytes: bytes}
}

func atlasUploadTileScene(tile gpui.AtlasTile) *gpui.Scene {
	bounds := gpui.Bounds[gpui.ScaledPixels]{
		Size: gpui.NewSize(gpui.ScaledPixels(float32(tile.Bounds.Size.Width)), gpui.ScaledPixels(float32(tile.Bounds.Size.Height))),
	}
	contentMask := gpui.ContentMask[gpui.ScaledPixels]{Bounds: bounds}
	scene := &gpui.Scene{}
	switch tile.TextureID.Kind {
	case gpui.AtlasTextureKindMonochrome:
		scene.InsertPrimitive(gpui.MonochromeSprite{
			Animation:      gpui.SpriteAnimationIDNone,
			Tile:           tile,
			Bounds:         bounds,
			ContentMask:    contentMask,
			Color:          gpui.Hsla{H: 0.0, S: 0.0, L: 1.0, A: 1.0},
			Transformation: gpui.DefaultTransformationMatrix(),
		})
	case gpui.AtlasTextureKindPolychrome:
		scene.InsertPrimitive(gpui.PolychromeSprite{
			Tile:        tile,
			Grayscale:   gpui.PaddedBool32FromBool(false),
			Opacity:     1.0,
			Bounds:      bounds,
			ContentMask: contentMask,
		})
	default:
		panic("internal error: entered unreachable code")
	}
	scene.Finish()
	return scene
}
