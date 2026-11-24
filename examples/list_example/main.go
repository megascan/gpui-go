package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

const (
	itemCount      = 40
	scrollbarWidth = 12.
)

// BottomListDemo shows a bottom-aligned list with a hand-drawn scrollbar.
type BottomListDemo struct {
	listState gpui.ListState
}

// NewBottomListDemo creates the demo.
func NewBottomListDemo() *BottomListDemo {
	return &BottomListDemo{
		listState: gpui.NewListState(itemCount, gpui.ListAlignmentBottom, gpui.Px(500.)).MeasureAll(),
	}
}

func pick(cond bool, a, b gpui.Rgba) gpui.Rgba {
	if cond {
		return a
	}
	return b
}

// Render renders the view.
func (d *BottomListDemo) Render(_ *gpui.Window, _ *gpui.Context[BottomListDemo]) gpui.IntoElement {
	maxOffset := d.listState.MaxOffsetForScrollbar().Y
	currentOffset := -d.listState.ScrollPxOffsetForScrollbar().Y

	viewportHeight := d.listState.ViewportBounds().Size.Height

	var rawFraction float32
	if maxOffset > gpui.Px(0.) {
		rawFraction = float32(currentOffset / maxOffset)
	}

	totalHeight := viewportHeight + maxOffset
	thumbHeight := gpui.Px(30.)
	if totalHeight > gpui.Px(0.) {
		thumbHeight = max(gpui.Px(viewportHeight.AsF32()*viewportHeight.AsF32()/totalHeight.AsF32()), gpui.Px(30.))
	}

	trackSpace := viewportHeight - thumbHeight
	thumbTop := trackSpace * gpui.Pixels(rawFraction)

	bugDetected := rawFraction > 1.0

	status := "OK: fraction <= 1.0 — thumb is within track."
	if bugDetected {
		status = fmt.Sprintf("BUG: fraction is %.3f (> 1.0) — thumb is off-track!", rawFraction)
	}

	return gpui.NewDiv().
		SizeFull().
		Bg(gpui.Rgb(0xFFFFFF)).
		Flex().
		FlexCol().
		P4().
		Gap2().
		Child(
			gpui.NewDiv().
				TextSm().
				Flex().
				FlexCol().
				Gap1().
				ChildText(fmt.Sprintf(
					"offset: %.0f / max: %.0f | fraction: %.3f",
					currentOffset.AsF32(),
					maxOffset.AsF32(),
					rawFraction,
				)).
				Child(
					gpui.NewDiv().
						TextColor(pick(bugDetected, gpui.Rgb(0xCC0000), gpui.Rgb(0x008800))).
						ChildText(status),
				),
		).
		Child(
			gpui.NewDiv().
				Flex1().
				Flex().
				FlexRow().
				OverflowHidden().
				Border1().
				BorderColor(gpui.Rgb(0xCCCCCC)).
				RoundedSm().
				Child(
					gpui.NewList(d.listState, func(index int, _ *gpui.Window, _ *gpui.App) gpui.AnyElement {
						height := gpui.Px(30. + float32(index%5)*10.)
						return gpui.NewDiv().
							H(height).
							WFull().
							Flex().
							ItemsCenter().
							Px3().
							BorderB1().
							BorderColor(gpui.Rgb(0xEEEEEE)).
							Bg(pick(index%2 == 0, gpui.Rgb(0xFAFAFA), gpui.Rgb(0xFFFFFF))).
							TextSm().
							ChildText(fmt.Sprintf("Item %d", index)).
							IntoAnyElement()
					}).
						Flex1(),
				).
				Child(
					gpui.NewDiv().
						W(gpui.Px(scrollbarWidth)).
						HFull().
						FlexShrink0().
						Bg(gpui.Rgb(0xE0E0E0)).
						Relative().
						Child(
							gpui.NewDiv().
								Absolute().
								Top(thumbTop).
								WFull().
								H(thumbHeight).
								Bg(pick(bugDetected, gpui.Rgb(0xCC0000), gpui.Rgb(0x888888))).
								RoundedSm(),
						),
				),
		)
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(400.), gpui.Px(500.)), cx)
		options := gpui.DefaultWindowOptions()
		options.Focus = true
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[BottomListDemo] {
			return cx.New(func(*gpui.Context[BottomListDemo]) *BottomListDemo { return NewBottomListDemo() })
		}); err != nil {
			panic(err)
		}
		cx.Activate(true)
	})
}

func main() {
	runExample()
}
