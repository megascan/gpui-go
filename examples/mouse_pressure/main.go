package main

import (
	"fmt"

	"github.com/megascan/gpui-go"
	"github.com/megascan/gpui-go/examples/internal/support"
	"github.com/megascan/gpui-go/platform"
)

// MousePressureExample shows the last mouse pressure event.
type MousePressureExample struct {
	pressureStage  gpui.PressureStage
	pressureAmount float32
}

// Render renders the view.
func (m *MousePressureExample) Render(_ *gpui.Window, cx *gpui.Context[MousePressureExample]) gpui.IntoElement {
	return gpui.NewDiv().
		Flex().
		FlexCol().
		Gap3().
		Bg(gpui.Rgb(0x505050)).
		Size(gpui.Px(500.0)).
		JustifyCenter().
		ItemsCenter().
		ShadowLg().
		Border1().
		BorderColor(gpui.Rgb(0x0000ff)).
		TextXl().
		TextColor(gpui.Rgb(0xffffff)).
		ChildText(fmt.Sprintf("Pressure stage: %s", m.pressureStage)).
		ChildText(fmt.Sprintf("Pressure amount: %.2f", m.pressureAmount)).
		OnMousePressure(cx.Listener((*MousePressureExample).onMousePressure))
}

func (m *MousePressureExample) onMousePressure(pressureEvent *gpui.MousePressureEvent, _ *gpui.Window, cx *gpui.Context[MousePressureExample]) {
	m.pressureAmount = pressureEvent.Pressure
	m.pressureStage = pressureEvent.Stage

	cx.Notify()
}

func runExample() {
	platform.Application().Run(func(cx *gpui.App) {
		if !support.LoadFonts(cx) {
			return
		}
		bounds := gpui.BoundsCentered(gpui.None[gpui.DisplayID](), gpui.NewSize(gpui.Px(500.), gpui.Px(500.0)), cx)

		options := gpui.DefaultWindowOptions()
		options.WindowBounds = gpui.Some(gpui.WindowBoundsWindowed(bounds))
		if _, err := cx.OpenWindow(options, func(_ *gpui.Window, cx *gpui.App) gpui.Entity[MousePressureExample] {
			return cx.New(func(_ *gpui.Context[MousePressureExample]) *MousePressureExample {
				return &MousePressureExample{
					pressureStage:  gpui.PressureStageZero,
					pressureAmount: 0.0,
				}
			})
		}); err != nil {
			panic(err)
		}

		cx.Activate(true)
	})
}

func main() {
	runExample()
}
