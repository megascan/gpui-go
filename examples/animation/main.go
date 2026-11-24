package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// Assets loads assets from the file system.
type Assets struct{}

// Load reads the file at path.
func (Assets) Load(path string) ([]byte, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false, fmt.Errorf("loading asset %s: %w", path, err)
	}
	return data, true, nil
}

// List lists the paths under path.
func (Assets) List(path string) ([]gpui.SharedString, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("listing assets %s: %w", path, err)
	}
	names := make([]gpui.SharedString, 0, len(entries))
	for _, entry := range entries {
		names = append(names, gpui.SharedString(filepath.Join(path, entry.Name())))
	}
	return names, nil
}

var arrowCircleSvg = func() gpui.SharedString {
	_, file, _, _ := runtime.Caller(0)
	return gpui.SharedString(filepath.Join(filepath.Dir(file), "image", "arrow_circle.svg"))
}()

type dampingDrag struct {
	startPosition gpui.Pixels
	startDamping  float32
}

// AnimationExample is the animation example view.
type AnimationExample struct {
	springPhase       uint8
	springDamping     float32
	springDampingDrag gpui.Option[dampingDrag]
}

const (
	minimumDamping float32 = 2.0
	maximumDamping float32 = 32.0
)

var dampingSliderWidth = gpui.Px(224.0)

func clampF32(v, lo, hi float32) float32 {
	return max(lo, min(hi, v))
}

// Render renders the view.
func (e *AnimationExample) Render(_ *gpui.Window, cx *gpui.Context[AnimationExample]) gpui.IntoElement {
	springPhase := e.springPhase
	springPosition := gpui.Px(98.0 * float32(springPhase))
	springDamping := e.springDamping
	spring := gpui.NewSpringConfig(170.0, springDamping, 1.0)
	_, dampingRatio := spring.Canonical()
	dampingFraction := (springDamping - minimumDamping) / (maximumDamping - minimumDamping)

	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		Bg(gpui.White()).
		TextColor(gpui.Black()).
		JustifyAround().
		Child(
			gpui.NewDiv().
				Flex().
				FlexCol().
				SizeFull().
				JustifyAround().
				Child(
					gpui.NewDiv().
						ID("content").
						Flex().
						FlexCol().
						H(gpui.Px(150.)).
						OverflowYScroll().
						WFull().
						Flex1().
						JustifyCenter().
						ItemsCenter().
						TextXl().
						Gap4().
						ChildText("Hello Animation").
						Child(
							gpui.NewDiv().
								ID("spring-demo").
								Flex().
								FlexCol().
								Gap2().
								P2().
								W(gpui.Px(240.0)).
								RoundedMd().
								Bg(gpui.NewRgba(0x00000010)).
								CursorPointer().
								OnMouseDown(gpui.MouseButtonLeft, cx.Listener(func(this *AnimationExample, _ *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[AnimationExample]) {
									this.springPhase = (this.springPhase + 1) % 3
									cx.Notify()
								})).
								ChildText(fmt.Sprintf("Target phase %d: click rapidly to redirect momentum", springPhase)).
								Child(
									gpui.NewDiv().
										ID("spring-damping-controls").
										Flex().
										FlexCol().
										Gap1().
										TextSm().
										OnClick(func(_ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.App) { cx.StopPropagation() }).
										ChildText(fmt.Sprintf("Drag damping: %.1f (ζ %.2f)", springDamping, dampingRatio)).
										Child(
											gpui.NewDiv().
												ID("spring-damping").
												Relative().
												H(gpui.Px(20.0)).
												W(dampingSliderWidth).
												CursorPointer().
												OnMouseDown(gpui.MouseButtonLeft, cx.Listener(func(this *AnimationExample, event *gpui.MouseDownEvent, _ *gpui.Window, cx *gpui.Context[AnimationExample]) {
													this.springDampingDrag = gpui.Some(dampingDrag{event.Position.X, this.springDamping})
													cx.StopPropagation()
												})).
												OnMouseMove(cx.Listener(func(this *AnimationExample, event *gpui.MouseMoveEvent, _ *gpui.Window, cx *gpui.Context[AnimationExample]) {
													drag, ok := this.springDampingDrag.Get()
													if !ok {
														return
													}
													delta := float32(event.Position.X-drag.startPosition) / float32(dampingSliderWidth)
													this.springDamping = clampF32(drag.startDamping+delta*(maximumDamping-minimumDamping), minimumDamping, maximumDamping)
													cx.StopPropagation()
													cx.Notify()
												})).
												OnMouseUp(gpui.MouseButtonLeft, cx.Listener(func(this *AnimationExample, _ *gpui.MouseUpEvent, _ *gpui.Window, cx *gpui.Context[AnimationExample]) {
													this.springDampingDrag = gpui.None[dampingDrag]()
													cx.StopPropagation()
												})).
												OnMouseUpOut(gpui.MouseButtonLeft, cx.Listener(func(this *AnimationExample, _ *gpui.MouseUpEvent, _ *gpui.Window, cx *gpui.Context[AnimationExample]) {
													this.springDampingDrag = gpui.None[dampingDrag]()
													cx.StopPropagation()
												})).
												Child(
													gpui.NewDiv().
														Absolute().
														Top(gpui.Px(8.0)).
														H(gpui.Px(4.0)).
														WFull().
														RoundedFull().
														Bg(gpui.NewRgba(0x00000030)),
												).
												Child(
													gpui.NewDiv().
														Absolute().
														Top(gpui.Px(8.0)).
														H(gpui.Px(4.0)).
														W(gpui.Relative(dampingFraction)).
														RoundedFull().
														Bg(gpui.NewRgba(0x3b82f6ff)),
												).
												Child(
													gpui.NewDiv().
														Absolute().
														Top(gpui.Px(3.0)).
														Left(gpui.Relative(dampingFraction)).
														Ml(gpui.Px(-7.0)).
														Size(gpui.Px(14.0)).
														RoundedFull().
														Bg(gpui.NewRgba(0x2563ebff)),
												),
										),
								).
								Child(
									gpui.NewDiv().Relative().H(gpui.Px(32.0)).WFull().Child(
										gpui.WithSpring(
											gpui.NewDiv().
												Absolute().
												Top1().
												Size(gpui.Px(24.0)).
												RoundedFull().
												Bg(gpui.NewRgba(0x3b82f6ff)).
												CursorPointer().
												DebugSelector(func() string { return "spring-position" }),
											"spring-position",
											gpui.NewSpringAnimation(spring).
												To(springPosition).
												WithEpsilon(0.25),
											func(this *gpui.Div, left gpui.Pixels) *gpui.Div { return this.Left(left) },
										),
									),
								).
								Child(
									gpui.WithSpring(
										gpui.NewDiv().H(gpui.Px(24.0)).RoundedSm(),
										"spring-phase",
										gpui.NewSpringAnimation(spring).
											To(gpui.AnimationPhase(float32(springPhase))).
											WithEpsilon(0.001),
										func(this *gpui.Div, phase gpui.AnimationPhase) *gpui.Div {
											var width gpui.Pixels
											var color gpui.Rgba
											if phase <= 1.0 {
												width = phase.InterpolateBetween[gpui.Pixels](0.0, 1.0, gpui.Px(48.0), gpui.Px(224.0))
												color = phase.InterpolateBetweenClamped[gpui.Rgba](0.0, 1.0, gpui.NewRgba(0xf97316ff), gpui.NewRgba(0x22c55eff))
											} else {
												width = phase.InterpolateBetween[gpui.Pixels](1.0, 2.0, gpui.Px(224.0), gpui.Px(96.0))
												color = phase.InterpolateBetweenClamped[gpui.Rgba](1.0, 2.0, gpui.NewRgba(0x22c55eff), gpui.NewRgba(0xa855f7ff))
											}
											return this.W(width).Bg(color)
										},
									),
								),
						).
						Child(
							gpui.WithAnimation(
								gpui.NewSvg().
									Size20().
									OverflowHidden().
									Path(arrowCircleSvg).
									TextColor(gpui.Black()),
								"image_circle",
								gpui.NewAnimation(2*time.Second).
									Repeat().
									WithEasing(gpui.Bounce(gpui.EaseInOut)),
								func(svg *gpui.Svg, delta float32) *gpui.Svg {
									return svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))
								},
							),
						),
				).
				Child(
					gpui.NewDiv().
						Flex().
						H(gpui.Px(64.)).
						WFull().
						P2().
						JustifyCenter().
						ItemsCenter().
						BorderT1().
						BorderColor(gpui.Black().Opacity(0.1)).
						Bg(gpui.Black().Opacity(0.05)).
						ChildText("Other Panel"),
				),
		)
}

func runExample() {
	platform.Application().WithAssets(Assets{}).Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(300.), gpui.Px(300.)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[AnimationExample] {
			cx.Activate(false)
			return cx.New(func(_ *gpui.Context[AnimationExample]) *AnimationExample {
				return &AnimationExample{
					springPhase:       0,
					springDamping:     14.0,
					springDampingDrag: gpui.None[dampingDrag](),
				}
			})
		}); err != nil {
			panic(fmt.Sprintf("called `Result::unwrap()` on an `Err` value: %v", err))
		}
	})
}

func main() {
	runExample()
}
