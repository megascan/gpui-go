package platform_test

import (
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"os"
	"runtime"
	"sync"
	"testing"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/platform"
)

var headlessRendererProbe = sync.OnceValue(func() error {
	renderer, err := platform.CurrentHeadlessRenderer()
	if err != nil {
		return err
	}
	if renderer == nil {
		return errors.New("the current platform has no headless renderer")
	}
	if destroyer, ok := renderer.(interface{ Destroy() }); ok {
		destroyer.Destroy()
	}
	return nil
})

func platformBenchOptions(b *testing.B, opts gpui.BenchOptions) gpui.BenchOptions {
	b.Helper()
	if err := headlessRendererProbe(); err != nil {
		b.Skipf("no GPU adapter: %v", err)
	}
	opts.TextSystem = platform.BenchTextSystem()
	opts.HeadlessRendererFactory = platform.CurrentHeadlessRenderer
	return opts
}

func sceneBatchCount(scene *gpui.Scene) int {
	count := 0
	for range scene.Batches() {
		count++
	}
	return count
}

func randomF32Range(rng *rand.Rand, start, end float32) float32 {
	return start + float32(rng.Float64())*(end-start)
}

func randomIntRange(rng *rand.Rand, start, end int) int {
	return start + rng.IntN(end-start)
}

type primitivesPattern uint8

const (
	primitivesPatternGrid primitivesPattern = iota
	primitivesPatternStripes
	primitivesPatternOverlap
	primitivesPatternNested
	primitivesPatternClusters
	primitivesPatternOverlay
	primitivesPatternRandom
)

type primitivesCase struct {
	pattern primitivesPattern
	seed    uint64
	count   int
}

func (c primitivesCase) String() string {
	var name string
	switch c.pattern {
	case primitivesPatternGrid:
		name = "grid"
	case primitivesPatternStripes:
		name = "stripes"
	case primitivesPatternOverlap:
		name = "overlap"
	case primitivesPatternNested:
		name = "nested"
	case primitivesPatternClusters:
		name = "clusters"
	case primitivesPatternOverlay:
		name = "overlay"
	case primitivesPatternRandom:
		return fmt.Sprintf("random-%d-seed-%d", c.count, c.seed)
	}
	return fmt.Sprintf("%s-%d", name, c.count)
}

func primitivesCases() []primitivesCase {
	seedIter, _ := gpui.CalculateSeeds(2, nil)
	var seeds []uint64
	for seed := range seedIter {
		seeds = append(seeds, seed)
	}
	var cases []primitivesCase
	for _, count := range []int{256, 1_024, 4_096} {
		for _, seed := range seeds {
			cases = append(cases, primitivesCase{pattern: primitivesPatternRandom, seed: seed, count: count})
		}
		for _, pattern := range []primitivesPattern{
			primitivesPatternGrid,
			primitivesPatternStripes,
			primitivesPatternOverlap,
			primitivesPatternNested,
			primitivesPatternClusters,
			primitivesPatternOverlay,
		} {
			cases = append(cases, primitivesCase{pattern: pattern, count: count})
		}
	}
	return cases
}

type primitivesKind uint8

const (
	primitivesKindQuad primitivesKind = iota
	primitivesKindShadow
	primitivesKindUnderline
	primitivesKindPath
	primitivesKindText
)

type primitivesRectangle struct {
	bounds     gpui.Bounds[gpui.Pixels]
	background gpui.Background
	color      gpui.Hsla
	kind       primitivesKind
	path       gpui.Path
	painted    gpui.PaintedPath
	text       gpui.SharedString
}

func randomPrimitivesRectangles(seed uint64, count int, extent float32) []primitivesRectangle {
	rng := rand.New(rand.NewPCG(seed, 0))
	smallest, largest := float32(math.Log(2)), float32(math.Log(float64(extent*0.6)))
	rectangles := make([]primitivesRectangle, 0, count)
	for range count {
		width := float32(math.Exp(float64(randomF32Range(rng, smallest, largest))))
		height := float32(math.Exp(float64(randomF32Range(rng, smallest, largest))))
		origin := gpui.NewPoint(
			gpui.Px(32.+randomF32Range(rng, 0.0, extent-width)),
			gpui.Px(32.+randomF32Range(rng, 0.0, extent-height)),
		)
		bounds := gpui.NewBounds(origin, gpui.NewSize(gpui.Px(width), gpui.Px(height)))
		color := gpui.NewHsla(randomF32Range(rng, 0.0, 1.0), 0.8, 0.5, 1.)
		rectangle := primitivesRectangle{bounds: bounds, background: color.ToBackground(), color: color}
		switch roll := randomIntRange(rng, 0, 20); {
		case roll < 9:
			rectangle.kind = primitivesKindQuad
		case roll < 11:
			rectangle.kind = primitivesKindShadow
		case roll < 13:
			rectangle.kind = primitivesKindUnderline
		case roll < 16:
			if path, ok := primitivesTriangle(bounds); ok {
				rectangle.kind = primitivesKindPath
				rectangle.painted = gpui.NewPaintedPath(path, 1., primitivesSceneContentMask(), color.ToBackground())
				rectangle.path = path
			} else {
				rectangle.kind = primitivesKindQuad
			}
		default:
			length := randomIntRange(rng, 1, 12)
			rectangle.kind = primitivesKindText
			rectangle.text = gpui.SharedString("Ag3 Wq7 fixture"[:length])
		}
		rectangles = append(rectangles, rectangle)
	}
	return rectangles
}

func primitivesTriangle(bounds gpui.Bounds[gpui.Pixels]) (gpui.Path, bool) {
	path := gpui.PathBuilderFill()
	path.MoveTo(bounds.Origin)
	path.LineTo(bounds.TopRight())
	path.LineTo(bounds.BottomLeft().Add(gpui.NewPoint(bounds.Size.Width/2., gpui.Px(0.))))
	path.Close()
	built, err := path.Build()
	return built, err == nil
}

func primitivesRectangles(c primitivesCase, extent float32) []primitivesRectangle {
	if c.pattern == primitivesPatternRandom {
		return randomPrimitivesRectangles(c.seed, c.count, extent)
	}
	columns := int(math.Ceil(math.Sqrt(float64(c.count))))
	cell := extent / float32(columns)
	stripeCount := int(math.Floor(float64(extent / 4.)))
	rectangles := make([]primitivesRectangle, 0, c.count)
	for index := range c.count {
		gridOrigin := gpui.NewPoint(
			gpui.Px(32.+float32(index%columns)*cell),
			gpui.Px(32.+float32(index/columns)*cell),
		)
		var bounds gpui.Bounds[gpui.Pixels]
		switch {
		case c.pattern == primitivesPatternGrid:
			bounds = gpui.NewBounds(gridOrigin, gpui.NewSize(gpui.Px(cell*0.4), gpui.Px(cell*0.4)))
		case c.pattern == primitivesPatternStripes:
			bounds = gpui.NewBounds(
				gpui.NewPoint(gpui.Px(32.), gpui.Px(32.+float32(index%stripeCount)*4.)),
				gpui.NewSize(gpui.Px(extent), gpui.Px(2.)),
			)
		case c.pattern == primitivesPatternOverlap:
			bounds = gpui.NewBounds(gpui.NewPoint(gpui.Px(32.), gpui.Px(32.)), gpui.NewSize(gpui.Px(extent), gpui.Px(extent)))
		case c.pattern == primitivesPatternNested:
			inset := extent * 0.3 * float32(index) / float32(c.count)
			bounds = gpui.NewBounds(
				gpui.NewPoint(gpui.Px(32.+inset), gpui.Px(32.+inset)),
				gpui.NewSize(gpui.Px(extent-2.*inset), gpui.Px(extent-2.*inset)),
			)
		case c.pattern == primitivesPatternClusters:
			origin := 32. + float32(index%2)*extent*0.55
			bounds = gpui.NewBounds(gpui.NewPoint(gpui.Px(origin), gpui.Px(origin)), gpui.NewSize(gpui.Px(extent*0.4), gpui.Px(extent*0.4)))
		case c.pattern == primitivesPatternOverlay && index%64 == 0:
			bounds = gpui.NewBounds(gpui.NewPoint(gpui.Px(32.), gpui.Px(32.)), gpui.NewSize(gpui.Px(extent), gpui.Px(extent)))
		default:
			bounds = gpui.NewBounds(gridOrigin, gpui.NewSize(gpui.Px(cell*0.4), gpui.Px(cell*0.4)))
		}
		color := gpui.NewHsla(float32(index)/float32(c.count), 1., 0.5, 1.)
		rectangles = append(rectangles, primitivesRectangle{
			bounds:     bounds,
			background: color.ToBackground(),
			color:      color,
			kind:       primitivesKindQuad,
		})
	}
	return rectangles
}

type orderingView struct {
	rectangles []primitivesRectangle
}

func (v *orderingView) Render(_ *gpui.Window, _ *gpui.Context[orderingView]) gpui.IntoElement {
	rectangles := v.rectangles
	return gpui.NewCanvas(
		func(gpui.Bounds[gpui.Pixels], *gpui.Window, *gpui.App) struct{} { return struct{}{} },
		func(_ gpui.Bounds[gpui.Pixels], _ struct{}, window *gpui.Window, cx *gpui.App) {
			for i := range rectangles {
				paintPrimitivesRectangle(&rectangles[i], window, cx)
			}
		},
	).SizeFull()
}

func paintPrimitivesRectangle(rectangle *primitivesRectangle, window *gpui.Window, cx *gpui.App) {
	switch rectangle.kind {
	case primitivesKindQuad:
		window.PaintQuad(gpui.NewFill(rectangle.bounds, rectangle.background))
	case primitivesKindShadow:
		window.PaintDropShadows(rectangle.bounds, gpui.CornersAll(gpui.Px(4.)), []gpui.BoxShadow{{
			Color:   rectangle.color,
			Offset:  gpui.NewPoint(gpui.Px(0.), gpui.Px(2.)),
			Blur:    gpui.Px(6.),
			Spread:  gpui.Px(0.),
			IsInset: false,
		}})
	case primitivesKindUnderline:
		window.PaintUnderline(rectangle.bounds.Origin, rectangle.bounds.Size.Width, gpui.UnderlineStyle{
			Thickness: min(rectangle.bounds.Size.Height, gpui.Px(4.)),
			Color:     gpui.Some(rectangle.color),
			Wavy:      false,
		})
	case primitivesKindPath:
		window.PaintPath(rectangle.path, rectangle.color.ToBackground())
	case primitivesKindText:
		fontSize := max(gpui.Px(8.), min(rectangle.bounds.Size.Height, gpui.Px(32.)))
		run := gpui.TextRun{
			Len:   len(rectangle.text),
			Font:  window.TextStyle().Font(),
			Color: rectangle.color,
		}
		line := window.TextSystem().ShapeLine(rectangle.text, fontSize, []gpui.TextRun{run}, gpui.None[gpui.Pixels]())
		if err := line.Paint(rectangle.bounds.Origin, fontSize, gpui.TextAlignLeft, gpui.None[gpui.Pixels](), window, cx); err != nil {
			fmt.Fprintf(os.Stderr, "failed to paint text: %v\n", err)
		}
	}
}

func primitivesGlyphTile(index int) gpui.AtlasTile {
	return gpui.AtlasTile{
		TextureID: gpui.AtlasTextureID{
			Index: uint32(index % 3),
			Kind:  gpui.AtlasTextureKindMonochrome,
		},
		TileID:  gpui.TileID(uint32(index)),
		Padding: 0,
		Bounds: gpui.NewBounds(
			gpui.NewPoint(gpui.DevicePixels(0), gpui.DevicePixels(0)),
			gpui.NewSize(gpui.DevicePixels(16), gpui.DevicePixels(16)),
		),
	}
}

func primitivesSceneContentMask() gpui.ContentMask[gpui.Pixels] {
	return gpui.ContentMask[gpui.Pixels]{
		Bounds: gpui.NewBounds(gpui.NewPoint(gpui.Px(0.), gpui.Px(0.)), gpui.NewSize(gpui.Px(1_024.), gpui.Px(1_024.))),
	}
}

func primitivesBuildScene(scene *gpui.Scene, rectangles []primitivesRectangle) {
	primitivesInsertPrimitives(scene, rectangles)
	scene.Finish()
}

func primitivesInsertPrimitives(scene *gpui.Scene, rectangles []primitivesRectangle) {
	scene.Clear()
	contentMask := primitivesSceneContentMask().Scale(1.)
	for index := range rectangles {
		rectangle := &rectangles[index]
		bounds := rectangle.bounds.Scale(1.)
		switch rectangle.kind {
		case primitivesKindQuad:
			scene.InsertPrimitive(gpui.Quad{
				Bounds:      bounds,
				ContentMask: contentMask,
				Background:  rectangle.background,
			})
		case primitivesKindShadow:
			scene.InsertPrimitive(gpui.Shadow{
				BlurRadius:       gpui.ScaledPixels(6.),
				Bounds:           bounds.Dilate(gpui.ScaledPixels(18.)),
				ContentMask:      contentMask,
				Color:            rectangle.color,
				ShapeBounds:      bounds,
				ShapeCornerRadii: gpui.CornersAll(gpui.ScaledPixels(4.)),
				Inset:            gpui.PaddedBool32FromBool(false),
			})
		case primitivesKindUnderline:
			scene.InsertPrimitive(gpui.Underline{
				Bounds:      bounds,
				ContentMask: contentMask,
				Color:       rectangle.color,
				Thickness:   min(bounds.Size.Height, gpui.ScaledPixels(4.)),
			})
		case primitivesKindPath:
			scene.InsertPrimitive(rectangle.painted)
		case primitivesKindText:
			glyphSize := min(bounds.Size.Height, gpui.ScaledPixels(32.))
			for glyph := range len(rectangle.text) {
				origin := bounds.Origin.Add(gpui.NewPoint(glyphSize*gpui.ScaledPixels(float32(glyph)*0.6), gpui.ScaledPixels(0.)))
				scene.InsertPrimitive(gpui.MonochromeSprite{
					Animation:      gpui.SpriteAnimationIDNone,
					Tile:           primitivesGlyphTile(index + glyph),
					Bounds:         gpui.NewBounds(origin, gpui.NewSize(glyphSize*0.6, glyphSize)),
					ContentMask:    contentMask,
					Color:          rectangle.color,
					Transformation: gpui.TransformationMatrixUnit(),
				})
			}
		}
	}
}

func BenchmarkPrimitives_SceneFinish(b *testing.B) {
	for _, c := range primitivesCases() {
		b.Run("Primitives/finish/"+c.String(), func(b *testing.B) {
			rectangles := primitivesRectangles(c, 512.)
			for b.Loop() {
				b.StopTimer()
				scene := &gpui.Scene{}
				primitivesInsertPrimitives(scene, rectangles)
				b.StartTimer()
				scene.Finish()
			}
		})
	}
}

func BenchmarkPrimitives_SceneOrdering(b *testing.B) {
	for _, c := range primitivesCases() {
		var scene *gpui.Scene
		b.Run("Primitives/scene/"+c.String(), func(b *testing.B) {
			rectangles := primitivesRectangles(c, 512.)
			scene = &gpui.Scene{}
			for b.Loop() {
				primitivesBuildScene(scene, rectangles)
				runtime.KeepAlive(scene)
			}
		})
		if os.Getenv("GPUI_BENCH_SCENE_STATS") == "1" && scene != nil {
			fmt.Fprintf(os.Stderr, "scene stats Primitives/scene/%v: batches=%d\n", c, sceneBatchCount(scene))
		}
	}
}

func BenchmarkPrimitives_PaintRectangles(b *testing.B) {
	opts := platformBenchOptions(b, gpui.BenchOptions{InputName: "case", Group: "Primitives/paint", Fps: 120})
	gpui.RunBenchInputs(b, opts, primitivesCases(), func(c primitivesCase, cx *gpui.BenchAppContext) {
		window := cx.AddEmptyWindow()
		extent := window.UpdateR(func(w *gpui.Window, _ *gpui.App) float32 {
			viewport := w.ViewportSize()
			return min(min(viewport.Width.AsF32(), viewport.Height.AsF32())-64., 512.)
		})
		rectangles := primitivesRectangles(c, extent)
		root := window.UpdateR(func(w *gpui.Window, app *gpui.App) gpui.Entity[orderingView] {
			return w.ReplaceRoot(app, func(*gpui.Window, *gpui.Context[orderingView]) *orderingView {
				return &orderingView{rectangles: rectangles}
			})
		})
		cx.RunUntilIdle()
		window.Update(func(w *gpui.Window, _ *gpui.App) { w.Refresh() })
		cx.RunUntilIdle()
		cx.BenchRenderer(root, func(_ *orderingView, _ *gpui.Window, cx *gpui.Context[orderingView]) { cx.Notify() })
		window.Update(func(w *gpui.Window, _ *gpui.App) {
			if os.Getenv("GPUI_BENCH_SCENE_STATS") == "1" {
				fmt.Fprintf(os.Stderr, "scene stats Primitives/paint/%v: batches=%d\n", c, sceneBatchCount(w.RenderedScene()))
			}
		})
		window.Update(func(w *gpui.Window, _ *gpui.App) { w.RemoveWindow() })
		cx.RunUntilIdle()
	})
}
