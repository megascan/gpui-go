//go:build linux

package wayland

import (
	"slices"
	"testing"

	gpui "github.com/megascan/gpui-go"
)

func TestWindowPresentationStateTests_FailureTracksWhetherTheSurfaceHasPresented(t *testing.T) {
	cases := []struct{ from, want presentationState }{
		{presentationStateUnpresented, presentationStateRetryBeforeFirstPresent},
		{presentationStateRetryBeforeFirstPresent, presentationStateRetryBeforeFirstPresent},
		{presentationStatePresented, presentationStateRetryAfterPresent},
		{presentationStateRetryAfterPresent, presentationStateRetryAfterPresent},
	}
	for _, c := range cases {
		if got := c.from.failed(); got != c.want {
			t.Errorf("%d.failed() = %d, want %d", c.from, got, c.want)
		}
	}
}

func TestWindowPresentationStateTests_OnlyRetryStatesRequirePresentation(t *testing.T) {
	if presentationStateUnpresented.requiresPresentation() {
		t.Error("Unpresented requires presentation")
	}
	if presentationStatePresented.requiresPresentation() {
		t.Error("Presented requires presentation")
	}
	if !presentationStateRetryBeforeFirstPresent.requiresPresentation() {
		t.Error("RetryBeforeFirstPresent does not require presentation")
	}
	if !presentationStateRetryAfterPresent.requiresPresentation() {
		t.Error("RetryAfterPresent does not require presentation")
	}
}

type rect struct{ x, y, width, height int32 }

func rectangles(area gpui.Bounds[int32], radii gpui.Corners[gpui.Pixels]) []rect {
	var out []rect
	addRoundedRect(area, radii, func(x, y, width, height int32) {
		out = append(out, rect{x, y, width, height})
	})
	return out
}

func TestWindowRoundedRectTests_SquareCorners(t *testing.T) {
	area := gpui.NewBounds(gpui.NewPoint[int32](10, 20), gpui.NewSize[int32](8, 8))
	got := rectangles(area, gpui.Corners[gpui.Pixels]{})
	if want := []rect{{10, 20, 8, 8}}; !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWindowRoundedRectTests_IndependentCorners(t *testing.T) {
	area := gpui.NewBounds(gpui.NewPoint[int32](10, 20), gpui.NewSize[int32](8, 8))
	radii := gpui.Corners[gpui.Pixels]{
		TopLeft:     gpui.Px(4.0),
		TopRight:    gpui.Px(0.0),
		BottomLeft:  gpui.Px(0.0),
		BottomRight: gpui.Px(4.0),
	}
	want := []rect{
		{12, 20, 6, 1},
		{11, 21, 7, 1},
		{10, 22, 8, 1},
		{10, 23, 8, 1},
		{10, 27, 6, 1},
		{10, 26, 7, 1},
		{10, 25, 8, 1},
		{10, 24, 8, 1},
	}
	if got := rectangles(area, radii); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWindowRoundedRectTests_ClampsRadii(t *testing.T) {
	area := gpui.NewBounds(gpui.NewPoint[int32](10, 20), gpui.NewSize[int32](10, 6))
	if got, want := rectangles(area, gpui.CornersAll(gpui.Px(100.0))), rectangles(area, gpui.CornersAll(gpui.Px(3.0))); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if got, want := rectangles(area, gpui.CornersAll(gpui.Px(-1.0))), rectangles(area, gpui.Corners[gpui.Pixels]{}); !slices.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestWindowRoundedRectTests_EmptyArea(t *testing.T) {
	for _, dimensions := range []gpui.Size[int32]{gpui.NewSize[int32](0, 10), gpui.NewSize[int32](10, 0), gpui.NewSize[int32](-1, 10), gpui.NewSize[int32](10, -1)} {
		area := gpui.NewBounds(gpui.NewPoint[int32](10, 20), dimensions)
		if got := rectangles(area, gpui.CornersAll(gpui.Px(4.0))); len(got) != 0 {
			t.Fatalf("%v: got %v, want none", dimensions, got)
		}
	}
}
