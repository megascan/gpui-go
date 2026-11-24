//go:build windows

package windows

import (
	"testing"
	"time"

	gpui "github.com/megascan/gpui-go"
)

func TestWindow_DoubleClickInterval(t *testing.T) {
	state := newClickState()
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](0, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := state.update(gpui.MouseButtonRight, gpui.NewPoint[gpui.DevicePixels](0, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](0, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](0, 0)); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
	state.lastClick = state.lastClick.Add(-700 * time.Millisecond)
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](0, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
}

func TestWindow_DoubleClickSpatialTolerance(t *testing.T) {
	state := newClickState()
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](-3, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := state.update(gpui.MouseButtonLeft, gpui.NewPoint[gpui.DevicePixels](0, 3)); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
	if got := state.update(gpui.MouseButtonRight, gpui.NewPoint[gpui.DevicePixels](3, 2)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
	if got := state.update(gpui.MouseButtonRight, gpui.NewPoint[gpui.DevicePixels](10, 0)); got != 1 {
		t.Fatalf("got %d, want 1", got)
	}
}
