package gpui

import (
	"fmt"
	"testing"
	"time"
)

func TestAssets_EmptyRenderImageDoesNotPanic(t *testing.T) {
	image := NewRenderImage(nil)
	if image.FrameCount() != 0 {
		t.Fatalf("frame count %d", image.FrameCount())
	}
	if image.Size(0) != (Size[DevicePixels]{}) {
		t.Fatalf("size %v", image.Size(0))
	}
	if b, ok := image.AsBytes(0); ok || b != nil {
		t.Fatalf("bytes %v", b)
	}
	if image.renderSize(0) != (Size[Pixels]{}) {
		t.Fatalf("render size %v", image.renderSize(0))
	}
	if image.Delay(0) != 100*time.Millisecond {
		t.Fatalf("delay %v", image.Delay(0))
	}
	_ = fmt.Sprintf("%v", image)
}
