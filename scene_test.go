package gpui

import (
	"slices"
	"testing"
)

func sceneTestSquare(x float32) Bounds[ScaledPixels] {
	return Bounds[ScaledPixels]{
		Origin: NewPoint(ScaledPixels(x), ScaledPixels(0.)),
		Size:   NewSize(ScaledPixels(10.), ScaledPixels(10.)),
	}
}

func sceneTestQuad(x float32) Quad {
	return Quad{
		Bounds:      sceneTestSquare(x),
		ContentMask: ContentMask[ScaledPixels]{Bounds: sceneTestSquare(x)},
	}
}

func sceneTestUnderline(x float32) Underline {
	return Underline{
		Order:       0,
		Pad:         0,
		Bounds:      sceneTestSquare(x),
		ContentMask: ContentMask[ScaledPixels]{Bounds: sceneTestSquare(x)},
		Color:       Hsla{},
		Thickness:   ScaledPixels(1.),
		Wavy:        PaddedBool32FromBool(false),
	}
}

func sceneTestBatchLabels(scene *Scene) []string {
	scene.Finish()
	var labels []string
	for batch := range scene.Batches() {
		labels = append(labels, batch.Label())
	}
	return labels
}

func assertLabels(t *testing.T, got []string, want ...string) {
	t.Helper()
	if !slices.Equal(got, want) {
		t.Fatalf("batch labels = %q, want %q", got, want)
	}
}

func TestScene_ANativeViewInALayerIsCoveredByWhatTheLayerPaintsAfterIt(t *testing.T) {
	var scene Scene
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(0.))
	scene.InsertPrimitive(PaintNativeView{
		Order:       0,
		ID:          NativeViewID(1),
		HitboxID:    HitboxID(0),
		Bounds:      sceneTestSquare(0.),
		ContentMask: ContentMask[ScaledPixels]{Bounds: sceneTestSquare(0.)},
		Background:  Hsla{},
	})
	scene.InsertPrimitive(sceneTestQuad(0.))
	scene.InsertPrimitive(sceneTestQuad(20.))
	scene.PopLayer()
	assertLabels(t, sceneTestBatchLabels(&scene), "quads (1)", "native views (1)", "quads (2)")
}

func TestScene_ANativeViewPaintedTwiceIsKeptWhereItWasPaintedFirst(t *testing.T) {
	nativeView := func(hitboxID HitboxID, x float32) PaintNativeView {
		return PaintNativeView{
			Order:       0,
			ID:          NativeViewID(1),
			HitboxID:    hitboxID,
			Bounds:      sceneTestSquare(x),
			ContentMask: ContentMask[ScaledPixels]{Bounds: sceneTestSquare(x)},
			Background:  Hsla{},
		}
	}
	firstHitbox := HitboxID(0)
	secondHitbox := firstHitbox + 1
	var scene Scene
	scene.InsertPrimitive(sceneTestUnderline(0.))
	scene.InsertPrimitive(sceneTestQuad(0.))
	scene.InsertPrimitive(nativeView(firstHitbox, 0.))
	scene.InsertPrimitive(nativeView(secondHitbox, 20.))
	scene.Finish()

	var shown []Bounds[ScaledPixels]
	for _, nativeView := range scene.NativeViews {
		shown = append(shown, nativeView.Bounds)
	}
	if want := []Bounds[ScaledPixels]{sceneTestSquare(0.)}; !slices.Equal(shown, want) {
		t.Fatalf("shown = %v, want %v", shown, want)
	}
}

func TestScene_LayersAreDrawnInPaintOrder(t *testing.T) {
	var scene Scene
	scene.InsertPrimitive(sceneTestQuad(0.))
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestUnderline(0.))
	scene.InsertPrimitive(sceneTestUnderline(5.))
	scene.PopLayer()
	scene.InsertPrimitive(sceneTestQuad(5.))
	assertLabels(t, sceneTestBatchLabels(&scene), "quads (1)", "underlines (2)", "quads (1)")

	scene = Scene{}
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestUnderline(0.))
	scene.PopLayer()
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(5.))
	scene.PopLayer()
	assertLabels(t, sceneTestBatchLabels(&scene), "underlines (1)", "quads (1)")
}

func TestScene_NestedLayersAreDrawnAboveTheirParents(t *testing.T) {
	var scene Scene
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestUnderline(0.))
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(5.))
	scene.PopLayer()
	scene.InsertPrimitive(sceneTestUnderline(5.))
	scene.PopLayer()
	assertLabels(t, sceneTestBatchLabels(&scene), "underlines (2)", "quads (1)")

	scene = Scene{}
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(0.))
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(5.))
	scene.PopLayer()
	scene.InsertPrimitive(sceneTestQuad(8.))
	scene.PopLayer()
	assertLabels(t, sceneTestBatchLabels(&scene), "quads (3)")
	var bounds []Bounds[ScaledPixels]
	for _, quad := range scene.Quads {
		bounds = append(bounds, quad.Bounds)
	}
	if want := []Bounds[ScaledPixels]{sceneTestSquare(0.), sceneTestSquare(8.), sceneTestSquare(5.)}; !slices.Equal(bounds, want) {
		t.Fatalf("quad bounds = %v, want %v", bounds, want)
	}

	scene = Scene{}
	scene.PushLayer()
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestUnderline(0.))
	scene.PopLayer()
	scene.PopLayer()
	scene.PushLayer()
	scene.InsertPrimitive(sceneTestQuad(5.))
	scene.PopLayer()
	assertLabels(t, sceneTestBatchLabels(&scene), "underlines (1)", "quads (1)")
}
