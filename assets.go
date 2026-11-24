package gpui

import (
	"fmt"
	"sync/atomic"
	"time"
)

// AssetSource is a source of assets for the app to use.
type AssetSource interface {
	// Load loads the asset at path; ok is false when there is none.
	Load(path string) (data []byte, ok bool, err error)
	// List lists the assets under path.
	List(path string) ([]SharedString, error)
}

// NoAssets is the AssetSource with no assets (upstream's impl for ()).
type NoAssets struct{}

// Load returns no asset.
func (NoAssets) Load(path string) ([]byte, bool, error) {
	return nil, false, nil
}

// List returns no assets.
func (NoAssets) List(path string) ([]SharedString, error) {
	return []SharedString{}, nil
}

// ImageID is a unique identifier for the image cache.
type ImageID uint

// RenderImageParams identifies a decoded image frame in the sprite atlas.
type RenderImageParams struct {
	// ImageID is the image.
	ImageID ImageID
	// FrameIndex is the frame of the image.
	FrameIndex int
}

// RenderImage is a cached and processed image, in BGRA format. Two images are equal when their
// IDs are.
type RenderImage struct {
	// ID is the ID associated with this image.
	ID          ImageID
	scaleFactor float32
	data        []ImageFrame
}

var nextRenderImageID atomic.Uint64

// NewRenderImage creates an image from the given frames.
func NewRenderImage(data []ImageFrame) *RenderImage {
	return &RenderImage{
		ID:          ImageID(nextRenderImageID.Add(1) - 1),
		scaleFactor: 1.0,
		data:        data,
	}
}

func (r *RenderImage) frame(frameIndex int) (*ImageFrame, bool) {
	if frameIndex < 0 || frameIndex >= len(r.data) {
		return nil, false
	}
	return &r.data[frameIndex], true
}

// AsBytes returns the BGRA bytes of a frame.
func (r *RenderImage) AsBytes(frameIndex int) ([]byte, bool) {
	f, ok := r.frame(frameIndex)
	if !ok {
		return nil, false
	}
	return f.Buffer.Pix, true
}

// Size returns the size of a frame in pixels; zero when there is no such frame.
func (r *RenderImage) Size(frameIndex int) Size[DevicePixels] {
	f, ok := r.frame(frameIndex)
	if !ok {
		return Size[DevicePixels]{}
	}
	b := f.Buffer.Bounds()
	return NewSize(DevicePixels(b.Dx()), DevicePixels(b.Dy()))
}

func (r *RenderImage) renderSize(frameIndex int) Size[Pixels] {
	s := r.Size(frameIndex)
	return NewSize(Pixels(float32(s.Width)/r.scaleFactor), Pixels(float32(s.Height)/r.scaleFactor))
}

// Delay returns the delay of a frame from the previous one; 100ms when there is no such frame.
func (r *RenderImage) Delay(frameIndex int) time.Duration {
	f, ok := r.frame(frameIndex)
	if !ok {
		return 100 * time.Millisecond
	}
	return f.Delay
}

// FrameCount returns the number of frames.
func (r *RenderImage) FrameCount() int {
	return len(r.data)
}

// String formats the image for debugging.
func (r *RenderImage) String() string {
	size := "None"
	if len(r.data) > 0 {
		b := r.data[0].Buffer.Bounds()
		size = fmt.Sprintf("Some((%d, %d))", b.Dx(), b.Dy())
	}
	return fmt.Sprintf("ImageData { id: ImageId(%d), size: %s }", r.ID, size)
}
