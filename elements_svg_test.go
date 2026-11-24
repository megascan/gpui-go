package gpui

import (
	"math"
	"testing"
	"time"
)

var svgTestSquareSvg = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16"><rect width="8" height="16"/></svg>`)

type svgTestAnimatedSvgView struct {
	transformation TimedTransformation
	renders        *int
}

func (v *svgTestAnimatedSvgView) Render(_ *Window, _ *Context[svgTestAnimatedSvgView]) IntoElement {
	*v.renders++
	return NewDiv().Child(
		NewSvg().
			Data(svgTestSquareSvg).
			Size(Px(16.)).
			TextColor(Red()).
			WithTimedTransformation(v.transformation),
	)
}

type svgTestPose struct {
	rotationScale [2][2]float32
	centerOffset  [2]float32
	alpha         float32
}

func svgTestAnimate(tcx *TestAppContext, transformation TimedTransformation) (func(advance time.Duration) svgTestPose, *int, func()) {
	renders := new(int)
	view, cx := tcx.AddWindowView(func(*Window, *Context[svgTestAnimatedSvgView]) *svgTestAnimatedSvgView {
		return &svgTestAnimatedSvgView{transformation: transformation, renders: renders}
	})
	cx.RunUntilParked()
	poseAfter := func(advance time.Duration) svgTestPose {
		tcx.Executor().AdvanceClock(advance)
		return cx.UpdateR(func(window *Window, cx *App) svgTestPose {
			window.SimulateNextFrame(cx)
			window.advanceSceneAnimations(cx)
			var sprite *MonochromeSprite
			for i := range window.renderedFrame.scene.MonochromeSprites {
				if window.renderedFrame.scene.MonochromeSprites[i].Animation != SpriteAnimationIDNone {
					sprite = &window.renderedFrame.scene.MonochromeSprites[i]
					break
				}
			}
			if sprite == nil {
				panic("the SVG should be painted as an animated sprite")
			}
			center := sprite.Bounds.Center()
			rs := sprite.Transformation.RotationScale
			a, b, c, d := rs[0][0], rs[0][1], rs[1][0], rs[1][1]
			x, y := sprite.Transformation.Translation[0], sprite.Transformation.Translation[1]
			return svgTestPose{
				rotationScale: rs,
				centerOffset: [2]float32{
					a*float32(center.X) + b*float32(center.Y) + x - float32(center.X),
					c*float32(center.X) + d*float32(center.Y) + y - float32(center.Y),
				},
				alpha: sprite.Color.A,
			}
		})
	}
	release := func() {
		view.Release()
		windowTestReleaseAll(tcx)
	}
	return poseAfter, renders, release
}

func svgTestAbs(v float32) float32 {
	return float32(math.Abs(float64(v)))
}

func TestSvg_RotationAdvancesWithoutRerendering(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		poseAfter, renders, release := svgTestAnimate(tcx, TimedTransformationRotation(time.Second, DefaultTransformation()))
		defer release()
		rendersBeforeFrames := *renders

		for quarter := range 4 {
			advance := 250 * time.Millisecond
			if quarter == 0 {
				advance = 0
			}
			pose := poseAfter(advance)
			angle := float64(float32(quarter) * float32(math.Pi/2))
			sin, cos := float32(math.Sin(angle)), float32(math.Cos(angle))
			expected := [2][2]float32{{cos, -sin}, {sin, cos}}
			for i := range 2 {
				for j := range 2 {
					if svgTestAbs(pose.rotationScale[i][j]-expected[i][j]) >= 1e-5 {
						t.Fatalf("quarter %d", quarter)
					}
				}
			}
			for _, offset := range pose.centerOffset {
				if svgTestAbs(offset) >= 1e-3 {
					t.Fatalf("quarter %d: center offset %v", quarter, pose.centerOffset)
				}
			}
		}
		if *renders != rendersBeforeFrames {
			t.Fatalf("renders = %d, want %d", *renders, rendersBeforeFrames)
		}
	})
}

func TestSvg_TimedTransformationEasesEachValueSeparately(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		poseAfter, renders, release := svgTestAnimate(tcx, NewTimedTransformation(time.Second).
			From(TransformationAlpha(0.4)).
			To(TransformationScale(NewSize[float32](1.5, 1.5)).WithAlpha(0.)).
			WithScaleEasing(EasingEaseOutQuint))
		defer release()
		rendersBeforeFrames := *renders

		pose := poseAfter(0)
		if svgTestAbs(pose.rotationScale[0][0]-1.) >= 1e-5 {
			t.Fatalf("scale = %v", pose.rotationScale[0][0])
		}
		if svgTestAbs(pose.alpha-0.4) >= 1e-5 {
			t.Fatalf("alpha = %v", pose.alpha)
		}

		pose = poseAfter(500 * time.Millisecond)
		half := float32(0.5)
		if svgTestAbs(pose.rotationScale[0][0]-(1.+0.5*(1.-half*half*half*half*half))) >= 1e-5 {
			t.Fatalf("scale = %v", pose.rotationScale[0][0])
		}
		if svgTestAbs(pose.alpha-0.2) >= 1e-5 {
			t.Fatalf("alpha = %v", pose.alpha)
		}
		if *renders != rendersBeforeFrames {
			t.Fatalf("renders = %d, want %d", *renders, rendersBeforeFrames)
		}
	})
}

func TestSvg_HeartbeatReturnsToItsStart(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		poseAfter, _, release := svgTestAnimate(tcx, NewTimedTransformation(time.Second).
			To(TransformationScale(NewSize[float32](1.12, 1.12))).
			WithScaleEasing(EasingHeartbeat))
		defer release()

		if v := poseAfter(0).rotationScale[0][0]; svgTestAbs(v-1.) >= 1e-3 {
			t.Fatalf("start = %v", v)
		}
		if v := poseAfter(200 * time.Millisecond).rotationScale[0][0]; svgTestAbs(v-1.12) >= 1e-3 {
			t.Fatalf("first beat = %v", v)
		}
		secondBeat := poseAfter(200 * time.Millisecond).rotationScale[0][0]
		if !(secondBeat > 1.05 && secondBeat < 1.09) {
			t.Fatalf("second beat = %v", secondBeat)
		}
		if v := poseAfter(599 * time.Millisecond).rotationScale[0][0]; svgTestAbs(v-1.) >= 1e-3 {
			t.Fatalf("end = %v", v)
		}
	})
}

type svgTestCappedView struct {
	spin TimedTransformation
}

func (v *svgTestCappedView) Render(_ *Window, _ *Context[svgTestCappedView]) IntoElement {
	icon := func() *Svg { return NewSvg().Data(svgTestSquareSvg).Size(Px(16.)).TextColor(Red()) }
	return NewDiv().
		Child(icon().WithTimedTransformation(v.spin.WithMaxFps(10.))).
		Child(icon().WithTimedTransformation(v.spin.WithMaxFps(20.))).
		Child(icon().WithTransformation(TransformationAlpha(0.5)))
}

func TestSvg_ScenePresentsAsOftenAsItsLeastCappedAnimation(t *testing.T) {
	RunTest(t, func(tc *TestRun) {
		tcx := tc.Cx()
		spin := TimedTransformationRotation(time.Second, DefaultTransformation())
		view, cx := tcx.AddWindowView(func(*Window, *Context[svgTestCappedView]) *svgTestCappedView {
			return &svgTestCappedView{spin: spin}
		})
		defer func() {
			view.Release()
			windowTestReleaseAll(tcx)
		}()
		cx.RunUntilParked()
		cx.Update(func(window *Window, _ *App) {
			interval, ok := window.renderedFrame.scene.animationFrameInterval()
			if !ok || interval != 50*time.Millisecond {
				t.Fatalf("interval = %v, %v; want Some(50ms)", interval, ok)
			}
		})
	})
}
