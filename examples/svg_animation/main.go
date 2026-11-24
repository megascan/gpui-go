// Command svg_animation compares SVGs animated with Svg.WithTimedTransformation, which moves
// them in the already painted scene, against the same SVGs animated by re-rendering with
// WithAnimation. It shows spinners and pulsing rings.
//
// The header counts how many times the view has rendered. With the retained scene the count
// stays put while the SVGs keep animating; when re-rendering it climbs once per frame. Pass
// `--rerender` to start in that mode, and `--max-fps <n>` to cap how often the retained scene is
// presented.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"time"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

var arrowCircleSvg = func() gpui.SharedString {
	_, file, _, _ := runtime.Caller(0)
	return gpui.SharedString(filepath.Join(filepath.Dir(file), "arrow_circle.svg"))
}()

var ringSvg = []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 16 16"><circle cx="8" cy="8" r="5" fill="none" stroke="black" stroke-width="1.5"/></svg>`)

const (
	spinnerCount = 100
	pulseCount   = 100
	spinPeriod   = 2 * time.Second
	pulsePeriod  = 1750 * time.Millisecond
)

// Mode is how the SVGs are animated.
type Mode uint8

const (
	// ModeRetainedScene animates in the retained scene.
	ModeRetainedScene Mode = iota
	// ModeRerender animates by re-rendering every frame.
	ModeRerender
)

// SvgAnimationExample is the svg_animation example view.
type SvgAnimationExample struct {
	mode        Mode
	maxFps      gpui.Option[float32]
	renderCount uint64
}

func (e *SvgAnimationExample) timed(transformation gpui.TimedTransformation) gpui.TimedTransformation {
	if maxFps, ok := e.maxFps.Get(); ok {
		return transformation.WithMaxFps(maxFps)
	}
	return transformation
}

func (e *SvgAnimationExample) pulse(index int) gpui.IntoElement {
	ring := gpui.NewSvg().Size4().Data(ringSvg).TextColor(gpui.Rgb(0x16a34a))
	var ringElement gpui.AnyElement
	switch e.mode {
	case ModeRetainedScene:
		ringElement = ring.
			WithTimedTransformation(
				e.timed(
					gpui.NewTimedTransformation(pulsePeriod).
						From(gpui.TransformationAlpha(0.6)).
						To(gpui.TransformationScale(gpui.NewSize[float32](1.5, 1.5)).WithAlpha(0.)).
						WithScaleEasing(gpui.EasingEaseOutQuint),
				),
			).
			IntoAnyElement()
	case ModeRerender:
		ringElement = gpui.WithAnimation(
			ring,
			gpui.ElementIDNamedInteger("pulse", uint64(index)),
			gpui.NewAnimation(pulsePeriod).RepeatSynced(),
			func(svg *gpui.Svg, delta float32) *gpui.Svg {
				scale := 1. + 0.5*gpui.EaseOutQuint()(delta)
				return svg.WithTransformation(
					gpui.TransformationScale(gpui.NewSize(scale, scale)).
						WithAlpha(0.6 * (1. - delta)),
				)
			},
		).IntoAnyElement()
	}
	return gpui.NewDiv().
		Size6().
		Flex().
		ItemsCenter().
		JustifyCenter().
		Child(gpui.NewDiv().Absolute().Size1p5().RoundedFull().Bg(gpui.Rgb(0x16a34a))).
		Child(ringElement)
}

func (e *SvgAnimationExample) spinner(index int) gpui.IntoElement {
	spinner := gpui.NewSvg().
		Size6().
		ExternalPath(arrowCircleSvg).
		TextColor(gpui.Rgb(0x3b82f6))
	switch e.mode {
	case ModeRetainedScene:
		return spinner.
			WithTimedTransformation(e.timed(gpui.TimedTransformationRotation(
				spinPeriod,
				gpui.DefaultTransformation(),
			))).
			IntoAnyElement()
	default:
		return gpui.WithAnimation(
			spinner,
			gpui.ElementIDNamedInteger("spinner", uint64(index)),
			gpui.NewAnimation(spinPeriod).RepeatSynced(),
			func(svg *gpui.Svg, delta float32) *gpui.Svg {
				return svg.WithTransformation(gpui.TransformationRotate(gpui.NewPercentage(delta).ToRadians()))
			},
		).IntoAnyElement()
	}
}

// Render renders the view.
func (e *SvgAnimationExample) Render(_ *gpui.Window, cx *gpui.Context[SvgAnimationExample]) gpui.IntoElement {
	e.renderCount++
	mode := e.mode
	modeLabel := "re-render (with_animation)"
	if mode == ModeRetainedScene {
		modeLabel = "retained scene (with_timed_transformation)"
	}

	spinners := make([]gpui.IntoElement, 0, spinnerCount)
	for index := range spinnerCount {
		spinners = append(spinners, e.spinner(index))
	}
	pulses := make([]gpui.IntoElement, 0, pulseCount)
	for index := range pulseCount {
		pulses = append(pulses, e.pulse(index))
	}

	return gpui.NewDiv().
		Flex().
		FlexCol().
		SizeFull().
		Bg(gpui.White()).
		TextColor(gpui.Black()).
		P4().
		Gap3().
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				ItemsCenter().
				Gap4().
				Child(
					gpui.NewDiv().
						ID("toggle-mode").
						Px3().
						Py1().
						RoundedMd().
						Bg(gpui.Rgb(0xe5e7eb)).
						CursorPointer().
						ChildText(fmt.Sprintf("Mode: %s", modeLabel)).
						OnClick(cx.Listener(func(this *SvgAnimationExample, _ *gpui.ClickEvent, _ *gpui.Window, cx *gpui.Context[SvgAnimationExample]) {
							if this.mode == ModeRetainedScene {
								this.mode = ModeRerender
							} else {
								this.mode = ModeRetainedScene
							}
							cx.Notify()
						})),
				).
				ChildText(fmt.Sprintf("Renders: %d", e.renderCount)),
		).
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				FlexWrap().
				Gap2().
				Children(spinners...),
		).
		Child(
			gpui.NewDiv().
				Flex().
				FlexRow().
				FlexWrap().
				Gap2().
				Children(pulses...),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(gpui.BoundsCentered(
			gpui.None[gpui.DisplayID](),
			gpui.NewSize(gpui.Px(720.), gpui.Px(480.)),
			cx,
		)))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[SvgAnimationExample] {
			cx.Activate(false)
			mode := ModeRetainedScene
			if slices.Contains(os.Args, "--rerender") {
				mode = ModeRerender
			}
			maxFps := gpui.None[float32]()
			if i := slices.Index(os.Args, "--max-fps"); i >= 0 && i+1 < len(os.Args) {
				if v, err := strconv.ParseFloat(os.Args[i+1], 32); err == nil {
					maxFps = gpui.Some(float32(v))
				}
			}
			return cx.New(func(_ *gpui.Context[SvgAnimationExample]) *SvgAnimationExample {
				return &SvgAnimationExample{
					mode:        mode,
					maxFps:      maxFps,
					renderCount: 0,
				}
			})
		}); err != nil {
			panic(fmt.Sprintf("failed to open window: %v", err))
		}
	})
}

func main() {
	runExample()
}
