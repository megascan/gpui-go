package gpui

import (
	"image"
	"testing"
)

const imgTestImgID = "test-img"

func imgTestImage(frameCount int) *RenderImage {
	frames := make([]ImageFrame, 0, frameCount)
	for range frameCount {
		frames = append(frames, newImageFrame(image.NewRGBA(image.Rect(0, 0, 1, 1))))
	}
	return NewRenderImage(frames)
}

func imgTestImageWithSize(width, height int) *RenderImage {
	return NewRenderImage([]ImageFrame{newImageFrame(image.NewRGBA(image.Rect(0, 0, width, height)))})
}

func imgTestSeedFrameIndex(frameIndex int) IntoElement {
	return NewCanvas(
		func(Bounds[Pixels], *Window, *App) struct{} { return struct{}{} },
		func(_ Bounds[Pixels], _ struct{}, window *Window, _ *App) {
			window.WithGlobalID(NewElementID(imgTestImgID), func(id *GlobalElementID, window *Window) struct{} {
				return window.WithElementState[imgState, struct{}](id, func(state Option[imgState], _ *Window) (struct{}, imgState) {
					s := state.Expect("img state should be initialized")
					s.frameIndex = frameIndex
					return struct{}{}, s
				})
			})
		},
	)
}

func imgTestDraw[RL, PP any](cx *VisualTestContext, origin Point[Pixels], size Size[Pixels], f func() Element[RL, PP]) {
	cx.Draw(origin, SizeAvailableSpaceFromPixels(size), func(*Window, *App) Element[RL, PP] { return f() })
}

func imgTestLastPolychromeSprite(cx *VisualTestContext, what string) (PolychromeSprite, float32) {
	return cx.UpdateR(func(window *Window, _ *App) PolychromeSprite {
		sprites := window.renderedFrame.scene.PolychromeSprites
		if len(sprites) == 0 {
			panic(what)
		}
		return sprites[len(sprites)-1]
	}), cx.UpdateR(func(window *Window, _ *App) float32 { return window.ScaleFactor() })
}

func TestImg_ZeroFrameImageDoesNotPanicOnPaint(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		imgTestDraw(tcx.AddEmptyWindow(), NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[ImgLayoutState, Option[Hitbox]] {
			return NewImg(ImageSourceRender{Image: imgTestImage(0)})
		})
	})
}

func TestImg_ImageObjectFitCoverCropsToElementBounds(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := tcx.AddEmptyWindow()
		img := imgTestImageWithSize(200, 100)
		imgTestDraw(window, NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[ImgLayoutState, Option[Hitbox]] {
			return NewImg(ImageSourceRender{Image: img}).SizeFull().ObjectFit(ObjectFitFill)
		})
		fill, _ := imgTestLastPolychromeSprite(window, "fill image should paint a sprite")
		fullTileBounds := fill.Tile.Bounds

		imgTestDraw(window, NewPoint(Px(10.), Px(20.)), NewSize(Px(100.), Px(100.)), func() Element[ImgLayoutState, Option[Hitbox]] {
			return NewImg(ImageSourceRender{Image: img}).SizeFull().ObjectFit(ObjectFitCover)
		})
		sprite, scaleFactor := imgTestLastPolychromeSprite(window, "cover image should paint a sprite")
		wantBounds := Bounds[ScaledPixels]{
			Origin: NewPoint(Px(10.).Scale(scaleFactor), Px(20.).Scale(scaleFactor)),
			Size:   NewSize(Px(100.).Scale(scaleFactor), Px(100.).Scale(scaleFactor)),
		}
		if sprite.Bounds != wantBounds {
			t.Fatalf("bounds = %v, want %v", sprite.Bounds, wantBounds)
		}
		tile := sprite.Tile.Bounds
		got := [4]int32{
			int32(tile.Origin.X) - int32(fullTileBounds.Origin.X),
			int32(tile.Origin.Y) - int32(fullTileBounds.Origin.Y),
			int32(tile.Size.Width),
			int32(tile.Size.Height),
		}
		if want := [4]int32{50, 0, 100, 100}; got != want {
			t.Fatalf("tile = %v, want %v", got, want)
		}
	})
}

func TestImg_ExplicitAspectRatioIsNotOverriddenByIntrinsicRatio(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := tcx.AddEmptyWindow()

		imgTestDraw(window, NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[DivFrameState, Option[Hitbox]] {
			return NewDiv().
				Size(Px(100.)).
				OverflowHidden().
				Child(
					NewImg(ImageSourceRender{Image: imgTestImageWithSize(100, 200)}).
						SizeFull().
						AspectSquare().
						ObjectFit(ObjectFitContain),
				)
		})

		sprite, scaleFactor := imgTestLastPolychromeSprite(window, "contained image should paint a sprite")
		want := Bounds[ScaledPixels]{
			Origin: NewPoint(Px(25.).Scale(scaleFactor), Px(0.).Scale(scaleFactor)),
			Size:   NewSize(Px(50.).Scale(scaleFactor), Px(100.).Scale(scaleFactor)),
		}
		if sprite.Bounds != want {
			t.Fatalf("bounds = %v, want %v", sprite.Bounds, want)
		}
	})
}

func TestImg_ImageObjectFitCoverClampsCornerRadiiToVisibleBounds(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := tcx.AddEmptyWindow()
		imgTestDraw(window, NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[ImgLayoutState, Option[Hitbox]] {
			return NewImg(ImageSourceRender{Image: imgTestImageWithSize(200, 100)}).
				SizeFull().
				Rounded(Px(100.)).
				ObjectFit(ObjectFitCover)
		})

		type radii struct {
			cornerRadius Option[ScaledPixels]
			expected     ScaledPixels
		}
		got := window.UpdateR(func(window *Window, _ *App) radii {
			sprites := window.renderedFrame.scene.PolychromeSprites
			r := radii{cornerRadius: None[ScaledPixels](), expected: Px(50.).Scale(window.ScaleFactor())}
			if len(sprites) != 0 {
				r.cornerRadius = Some(sprites[len(sprites)-1].CornerRadii.TopLeft)
			}
			return r
		})
		if got.cornerRadius != Some(got.expected) {
			t.Fatalf("corner radius = %v, want Some(%v)", got.cornerRadius, got.expected)
		}
	})
}

func TestImg_StaleFrameIndexIsClampedWhenImageChanges(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		window := tcx.AddEmptyWindow()

		imgTestDraw(window, NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[DivFrameState, Option[Hitbox]] {
			return NewDiv().
				Child(NewImg(ImageSourceRender{Image: imgTestImage(5)}).ID(imgTestImgID)).
				Child(imgTestSeedFrameIndex(4))
		})
		imgTestDraw(window, NewPoint(Px(0.), Px(0.)), NewSize(Px(100.), Px(100.)), func() Element[ImgLayoutState, Option[Hitbox]] {
			img := NewImg(ImageSourceRender{Image: imgTestImage(1)})
			img.ID(imgTestImgID)
			return img
		})
	})
}
